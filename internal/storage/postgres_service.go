package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gestor-documental/internal/buildinfo"
	"github.com/jackc/pgx/v5"
)

type postgresInstallation struct {
	Format        int    `json:"format"`
	Port          int    `json:"port"`
	AdminPassword string `json:"admin_password"`
	Password      string `json:"password"`
}

func postgresBinary(name string) (string, error) {
	if directory := os.Getenv("AIDEX_POSTGRES_BIN"); buildinfo.Channel == "source" && directory != "" {
		return filepath.Join(directory, name+executableSuffix()), nil
	}
	if executable, err := os.Executable(); err == nil {
		path := filepath.Join(filepath.Dir(executable), "tools", "postgresql", "bin", name+executableSuffix())
		if info, e := os.Stat(path); e == nil && info.Mode().IsRegular() {
			return path, nil
		}
	}
	// Development fallback. Production packages supply their own complete runtime.
	if path, err := exec.LookPath(name); buildinfo.Channel == "source" && err == nil {
		return path, nil
	}
	return "", fmt.Errorf("AIDEX PostgreSQL runtime is missing (%s)", name)
}
func executableSuffix() string {
	if os.PathSeparator == '\\' {
		return ".exe"
	}
	return ""
}
func postgresSecret() string {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		panic(err)
	}
	return hex.EncodeToString(value)
}
func postgresDSN(port int, user, password, database string) string {
	address := url.URL{Scheme: "postgresql", Host: net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), User: url.UserPassword(user, password), Path: "/" + database}
	address.RawQuery = "sslmode=disable&connect_timeout=10"
	return address.String()
}

func preparePostgres(ctx context.Context, state string) (string, string, func() error, error) {
	// The integration harness uses one real PostgreSQL server and an independent
	// schema per state directory, so hundreds of fixtures need no extra servers.
	if dsn := os.Getenv("AIDEX_TEST_POSTGRES_DSN"); buildinfo.Channel == "source" && dsn != "" {
		digest := sha256.Sum256([]byte(state))
		return dsn, "test_" + hex.EncodeToString(digest[:12]), nil, nil
	}
	directory := filepath.Join(state, "postgresql")
	if err := PrepareLocalPrivateDirectory(directory); err != nil {
		return "", "", nil, err
	}
	unlock, err := LockState(directory)
	if err != nil {
		return "", "", nil, err
	}
	// Retain ownership until all pools are closed and our server is stopped.
	fail := func(err error) (string, string, func() error, error) { unlock(); return "", "", nil, err }
	descriptor := filepath.Join(directory, "installation.json")
	var installation postgresInstallation
	contents, err := os.ReadFile(descriptor)
	if os.IsNotExist(err) {
		if entries, e := os.ReadDir(filepath.Join(directory, "data")); e == nil && len(entries) > 0 {
			return fail(fmt.Errorf("existing PostgreSQL data has no AIDEX ownership descriptor; data preserved"))
		}
		listener, e := net.Listen("tcp4", "127.0.0.1:0")
		if e != nil {
			return fail(e)
		}
		installation = postgresInstallation{Format: 1, Port: listener.Addr().(*net.TCPAddr).Port, Password: postgresSecret(), AdminPassword: postgresSecret()}
		listener.Close()
		contents, _ = json.Marshal(installation)
		file, e := os.OpenFile(descriptor, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return fail(e)
		}
		_, e = file.Write(contents)
		if e == nil {
			e = file.Sync()
		}
		closeErr := file.Close()
		if e != nil {
			return fail(e)
		}
		if closeErr != nil {
			return fail(closeErr)
		}
	} else if err != nil {
		return fail(err)
	} else {
		info, e := os.Lstat(descriptor)
		if e != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fail(fmt.Errorf("unsafe PostgreSQL installation descriptor"))
		}
		if e = json.Unmarshal(contents, &installation); e != nil {
			return fail(e)
		}
	}
	if installation.Format != 1 || installation.Port < 1024 || installation.Port > 65535 || len(installation.Password) != 64 || len(installation.AdminPassword) != 64 {
		return fail(fmt.Errorf("invalid PostgreSQL installation descriptor"))
	}
	if err = ProtectPrivatePath(descriptor, false); err != nil {
		return fail(err)
	}
	data := filepath.Join(directory, "data")
	if err = PrepareLocalPrivateDirectory(data); err != nil {
		return fail(err)
	}
	version, versionErr := os.ReadFile(filepath.Join(data, "PG_VERSION"))
	if os.IsNotExist(versionErr) {
		// Never initialize over unknown data or an interrupted initialization.
		if entries, e := os.ReadDir(data); e == nil && len(entries) > 0 {
			return fail(fmt.Errorf("incomplete PostgreSQL initialization; existing data preserved"))
		}
		passwordFile := filepath.Join(directory, ".init-password")
		passwordHandle, e := os.OpenFile(passwordFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return fail(e)
		}
		_, e = passwordHandle.WriteString(installation.AdminPassword + "\n")
		closeErr := passwordHandle.Close()
		if e != nil || closeErr != nil {
			os.Remove(passwordFile)
			return fail(fmt.Errorf("write private initialization credentials"))
		}
		binary, e := postgresBinary("initdb")
		if e != nil {
			os.Remove(passwordFile)
			return fail(e)
		}
		command := exec.CommandContext(ctx, binary, "-D", data, "--username=aidex_engine", "--pwfile="+passwordFile, "--auth=scram-sha-256", "--encoding=UTF8", "--locale=C", "--data-checksums")
		output, e := command.CombinedOutput()
		os.Remove(passwordFile)
		if e != nil {
			return fail(fmt.Errorf("initialize AIDEX PostgreSQL: %w: %s", e, output))
		}
	} else if versionErr != nil {
		return fail(versionErr)
	} else if strings.TrimSpace(string(version)) != "18" {
		return fail(fmt.Errorf("PostgreSQL major upgrade requires the matching AIDEX maintenance runtime"))
	}
	configuration := fmt.Sprintf("\n# Private AIDEX instance; never use the host's cluster.\nlisten_addresses = '127.0.0.1'\nport = %d\nunix_socket_directories = ''\npassword_encryption = 'scram-sha-256'\nmax_connections = 64\nshared_buffers = '128MB'\nfsync = on\nsynchronous_commit = on\nfull_page_writes = on\nlog_min_error_statement = 'panic'\nlog_parameter_max_length_on_error = 0\n", installation.Port)
	if info, e := os.Lstat(filepath.Join(data, "postgresql.auto.conf")); e != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return fail(fmt.Errorf("unsafe PostgreSQL configuration path"))
	}
	// postgresql.auto.conf belongs exclusively to this private cluster.
	if err = os.WriteFile(filepath.Join(data, "postgresql.auto.conf"), []byte(configuration), 0600); err != nil {
		return fail(err)
	}
	binary, err := postgresBinary("pg_ctl")
	if err != nil {
		return fail(err)
	}
	status := exec.CommandContext(ctx, binary, "-D", data, "status")
	if status.Run() != nil {
		output, e := exec.CommandContext(ctx, binary, "-D", data, "-l", filepath.Join(directory, "server.log"), "-w", "-t", "30", "start").CombinedOutput()
		if e != nil {
			return fail(fmt.Errorf("start AIDEX PostgreSQL: %w: %s", e, output))
		}
	}
	stopSupervisor := func() {}
	stop := func() error {
		stopSupervisor()
		stopCtx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		output, e := exec.CommandContext(stopCtx, binary, "-D", data, "-m", "fast", "-w", "-t", "30", "stop").CombinedOutput()
		unlock()
		if e != nil {
			return fmt.Errorf("stop AIDEX PostgreSQL: %w: %s", e, output)
		}
		return nil
	}
	admin, err := pgx.Connect(ctx, postgresDSN(installation.Port, "aidex_engine", installation.AdminPassword, "postgres"))
	if err != nil {
		stop()
		return "", "", nil, fmt.Errorf("connect private PostgreSQL administrator: %w", err)
	}
	var exists bool
	err = admin.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='aidex_app')").Scan(&exists)
	if err == nil && !exists {
		_, err = admin.Exec(ctx, "CREATE ROLE aidex_app LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD '"+installation.Password+"'")
	}
	if err == nil {
		err = admin.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname='aidex')").Scan(&exists)
	}
	if err == nil && !exists {
		_, err = admin.Exec(ctx, "CREATE DATABASE aidex OWNER aidex_app TEMPLATE template0 ENCODING 'UTF8' LC_COLLATE 'C' LC_CTYPE 'C'")
	}
	admin.Close(ctx)
	if err == nil {
		admin, err = pgx.Connect(ctx, postgresDSN(installation.Port, "aidex_engine", installation.AdminPassword, "aidex"))
		if err == nil {
			_, err = admin.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS unaccent WITH SCHEMA public; REVOKE CREATE ON SCHEMA public FROM PUBLIC; GRANT CREATE ON SCHEMA public TO aidex_app; REVOKE CONNECT ON DATABASE aidex FROM PUBLIC; GRANT CONNECT ON DATABASE aidex TO aidex_app;")
			admin.Close(ctx)
		}
	}
	if err != nil {
		stop()
		return "", "", nil, err
	}
	stopSupervisor = supervisePostgres(binary, data, filepath.Join(directory, "server.log"))
	return postgresDSN(installation.Port, "aidex_app", installation.Password, "aidex"), "aidex", stop, nil
}

// The AIDEX service owns recovery of its child engine. This also recovers a
// surviving cluster after the host restarts; the OS starts only the AIDEX host.
func supervisePostgres(binary, data, logPath string) func() {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			probeCtx, stop := context.WithTimeout(ctx, 5*time.Second)
			err := exec.CommandContext(probeCtx, binary, "-D", data, "status").Run()
			stop()
			if err == nil || ctx.Err() != nil {
				continue
			}
			restartCtx, stop := context.WithTimeout(ctx, 35*time.Second)
			err = exec.CommandContext(restartCtx, binary, "-D", data, "-l", logPath, "-w", "-t", "30", "start").Run()
			stop()
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				slog.Error("private PostgreSQL restart failed", "code", "POSTGRES_RESTART_FAILED")
			} else {
				slog.Info("private PostgreSQL recovered", "code", "POSTGRES_RESTARTED")
			}
		}
	}()
	return func() { cancel(); <-done }
}
