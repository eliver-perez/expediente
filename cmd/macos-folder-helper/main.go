//go:build darwin && cgo

// Auxiliar instalado root:wheel. Solo describe o autoriza un ticket local;
// no interpreta órdenes, rutas ni identidades suministradas como argumentos.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gestor-documental/internal/macosfolders"
)

var noncePattern = regexp.MustCompile(`^/[a-f0-9]{64}$`)
var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func ticketEndpoint(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "aidex-auth" || parsed.Host != "authorize" || parsed.User != nil || parsed.Fragment != "" || !noncePattern.MatchString(parsed.Path) || parsed.RawPath != "" {
		return "", fmt.Errorf("enlace de autorización no válido")
	}
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil || len(query) != 1 || len(query["port"]) != 1 {
		return "", fmt.Errorf("puerto de autorización no válido")
	}
	port, err := strconv.Atoi(query.Get("port"))
	if err != nil || port < 1 || port > 65535 || strconv.Itoa(port) != query.Get("port") {
		return "", fmt.Errorf("puerto de autorización no válido")
	}
	return "http://127.0.0.1:" + strconv.Itoa(port) + "/api/v1/local-folder-authorizations" + parsed.Path, nil
}

func call(ctx context.Context, endpoint, method string, result any) error {
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewBufferString("{}"))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("no se pudo contactar con el servicio local")
	}
	defer response.Body.Close()
	client.CloseIdleConnections()
	if response.StatusCode != 200 {
		return fmt.Errorf("el servicio rechazó la solicitud (HTTP %d); comprueba la sesión y genera otra autorización", response.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 16384)).Decode(result)
}

func consoleUser() (int, error) {
	info, err := os.Stat("/dev/console")
	if err != nil {
		return 0, err
	}
	uid := int(info.Sys().(*syscall.Stat_t).Uid)
	if uid < 500 {
		return 0, fmt.Errorf("abre la aplicación desde una sesión de usuario local de macOS")
	}
	return uid, nil
}

func run() error {
	if len(os.Args) == 2 && os.Args[1] == "preserve-archive" {
		uid, err := consoleUser()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		closeLog := configureAuthorizationLog()
		defer closeLog()
		count, err := macosfolders.ExposePreservedDocuments(ctx, uid, probeTraversal)
		if err != nil {
			return err
		}
		if count > 0 {
			if err = verifyFinder(ctx, uid, "/Library/Application Support/AIDEX/Documentos conservados"); err != nil {
				return err
			}
		}
		fmt.Printf("Documentos conservados: lectura para el usuario local de Finder; %d entradas.\n", count)
		return nil
	}
	if len(os.Args) == 3 && os.Args[1] == "check-traverse" {
		if os.Getuid() != os.Geteuid() || os.Geteuid() == 0 || !filepath.IsAbs(os.Args[2]) || filepath.Clean(os.Args[2]) != os.Args[2] {
			return fmt.Errorf("comprobación de recorrido no válida")
		}
		if err := syscall.Access(os.Args[2], 1); err != nil {
			if errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) {
				return errTraversalDenied
			}
			return err
		}
		return nil
	}
	if len(os.Args) == 3 && os.Args[1] == "check-read" {
		// Se utiliza únicamente después de bajar a la identidad local de Finder.
		// Esta operación no escribe ni eleva privilegios.
		if os.Geteuid() == 0 {
			return fmt.Errorf("la comprobación de Finder no puede ejecutarse como root")
		}
		if err := macosfolders.ValidateSelection(macosfolders.Selection{Path: os.Args[2], Source: "linked"}); err != nil && os.Args[2] != "/Library/Application Support/AIDEX/Documentos conservados" {
			return err
		}
		return checkRead(os.Args[2])
	}
	if len(os.Args) < 3 || len(os.Args) > 4 {
		return fmt.Errorf("operación de autorización no válida")
	}
	action := os.Args[1]
	if action != "describe" && action != "apply" {
		return fmt.Errorf("operación no permitida")
	}
	endpoint, err := ticketEndpoint(os.Args[2])
	if err != nil {
		return err
	}
	uid, err := consoleUser()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	var selection macosfolders.Selection
	if action == "describe" {
		if len(os.Args) != 3 || os.Getuid() != uid {
			return fmt.Errorf("la autorización debe abrirse en la sesión local activa")
		}
		if err = call(ctx, endpoint+"/describe", "GET", &selection); err != nil {
			return err
		}
		if err = macosfolders.ValidateSelection(selection); err != nil {
			return err
		}
		fmt.Println(macosfolders.Fingerprint(selection, uid))
		fmt.Println("Carpeta: " + filepath.Base(selection.Path))
		fmt.Println("Ubicación: " + selection.Path)
		if selection.Source == "linked" {
			fmt.Println("AIDEX podrá leer esta carpeta y su contenido. No podrá modificarlos.")
		} else {
			fmt.Println("AIDEX podrá leer, guardar y administrar documentos y subcarpetas. Tu usuario de Finder tendrá acceso de lectura, también a documentos nuevos.")
		}
		fmt.Println("Solo se autoriza esta carpeta; en las superiores únicamente se permite llegar hasta ella. No se concede acceso público ni se cambian los ajustes de privacidad.")
		return nil
	}
	if len(os.Args) != 4 || !digestPattern.MatchString(os.Args[3]) || os.Geteuid() != 0 {
		return fmt.Errorf("se requiere la confirmación administrativa de la aplicación instalada")
	}
	closeLog := configureAuthorizationLog()
	defer closeLog()
	slog.Info("folder authorization started", "real_uid", os.Getuid(), "effective_uid", os.Geteuid(), "effective_gid", os.Getegid())
	if err = verifyServiceEndpoint(ctx, endpoint); err != nil {
		slog.Error("folder authorization rejected", "operation", "verify_service_identity", "cause", err.Error())
		return err
	}
	if err = call(ctx, endpoint+"/claim", "POST", &selection); err != nil {
		return err
	}
	if macosfolders.Fingerprint(selection, uid) != os.Args[3] {
		return fmt.Errorf("cambió la selección o el usuario local; no se modificaron permisos")
	}
	account, err := user.Lookup("_aidex")
	if err != nil {
		return fmt.Errorf("no existe la cuenta de servicio _aidex")
	}
	serviceUID, err := strconv.Atoi(account.Uid)
	if err != nil || serviceUID < 400 || serviceUID >= 500 {
		return fmt.Errorf("identidad de servicio no válida")
	}
	slog.Info("folder authorization scope", "path", selection.Path, "source", selection.Source, "service_uid", serviceUID, "finder_uid", uid)
	count, applyErr := macosfolders.Authorize(ctx, selection, serviceUID, uid, probeTraversal)
	var checked struct {
		Accessible bool   `json:"accessible"`
		Code       string `json:"code"`
	}
	finishErr := call(ctx, endpoint+"/finish", "POST", &checked)
	if applyErr != nil {
		slog.Error("folder authorization incomplete", "operation", "apply_permissions", "path", selection.Path, "completed_entries", count, "cause", applyErr.Error())
		return fmt.Errorf("No se pudo completar la autorización. No se borró ningún documento. Consulta el registro de autorización para soporte")
	}
	slog.Info("folder authorization service check", "operation", "backend_effective_access", "path", selection.Path, "service_uid", serviceUID, "accessible", checked.Accessible, "code", checked.Code, "request_failed", finishErr != nil)
	if finishErr != nil {
		return fmt.Errorf("Permisos aplicados, pero no se pudo verificar el servicio: %w", finishErr)
	}
	if !checked.Accessible {
		if checked.Code == "ROOT_PRIVACY_DENIED" {
			return fmt.Errorf("macOS sigue bloqueando el acceso. Comprueba los permisos y, si es necesario, habilita aidex-server en Acceso total al disco. Después, comprueba nuevamente el acceso")
		}
		return fmt.Errorf("el servicio todavía no puede realizar la operación (diagnóstico %s). No se declara éxito; usa Comprobar acceso", checked.Code)
	}
	if selection.Source == "managed" {
		if err = verifyFinder(ctx, uid, selection.Path); err != nil {
			return err
		}
	}
	fmt.Printf("Carpeta autorizada. El servicio AIDEX verificó el acceso efectivo. Entradas procesadas: %d. Vuelve a AIDEX y usa Comprobar acceso.\n", count)
	return nil
}

func verifyFinder(ctx context.Context, uid int, path string) error {
	finder, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(finder.Gid)
	if err != nil {
		return err
	}
	groupIDs, err := finder.GroupIds()
	if err != nil {
		return err
	}
	groups := []uint32{}
	for _, id := range groupIDs {
		number, err := strconv.Atoi(id)
		if err != nil {
			return err
		}
		groups = append(groups, uint32(number))
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	probe := exec.CommandContext(ctx, self, "check-read", path)
	probe.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid), Groups: groups}}
	if err = probe.Run(); err != nil {
		slog.Error("folder authorization failed", "operation", "finder_effective_read", "path", path, "target_uid", uid, "effective_uid", os.Geteuid(), "cause", err.Error())
		return fmt.Errorf("la lectura con el usuario de Finder no pudo verificarse; revisa los permisos o los ajustes de privacidad de macOS")
	}
	return nil
}

func checkRead(path string) error {
	root, err := os.OpenRoot(path)
	if err != nil {
		return err
	}
	defer root.Close()
	return fs.WalkDir(root.FS(), ".", func(relative string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if path == "/Library/Application Support/AIDEX/Documentos conservados" {
				return nil
			}
			return fmt.Errorf("enlace no admitido")
		}
		file, err := root.Open(relative)
		if err != nil {
			return err
		}
		defer file.Close()
		if entry.IsDir() {
			return nil
		}
		_, err = file.Read(make([]byte, 1))
		if err == io.EOF {
			return nil
		}
		return err
	})
}

func registeredPID(output string) (string, error) {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[0] == "pid" && fields[1] == "=" {
			pid, err := strconv.Atoi(fields[2])
			if err == nil && pid > 1 {
				return strconv.Itoa(pid), nil
			}
		}
	}
	return "", fmt.Errorf("el servicio AIDEX no tiene un proceso activo")
}

func listenerMatches(output, pid string) bool {
	found := false
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		if line != pid {
			return false
		}
		found = true
	}
	return found
}

// El puerto del enlace no es autoridad: comprobar la configuración fija y que
// su listener pertenece realmente al proceso registrado en launchd. Una app
// local no puede suplantar al servicio para inventar tickets privilegiados.
func verifyServiceEndpoint(ctx context.Context, endpoint string) error {
	const data = "/Library/Application Support/AIDEX/data"
	for path := data; path != "/"; path = filepath.Dir(path) {
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("no se pudo validar la instalación local")
		}
	}
	root, err := os.OpenRoot(data)
	if err != nil {
		return fmt.Errorf("no se pudo abrir la configuración local")
	}
	defer root.Close()
	info, err := root.Lstat("config.json")
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("configuración local no válida")
	}
	file, err := root.Open("config.json")
	if err != nil {
		return fmt.Errorf("configuración local no disponible")
	}
	defer file.Close()
	after, err := file.Stat()
	if err != nil || !os.SameFile(info, after) {
		return fmt.Errorf("la configuración cambió durante la autorización")
	}
	var config struct {
		Listen string `json:"listen_address"`
	}
	if err = json.NewDecoder(io.LimitReader(file, 1<<20)).Decode(&config); err != nil {
		return fmt.Errorf("no se pudo validar el puerto configurado")
	}
	_, port, err := net.SplitHostPort(config.Listen)
	if err != nil {
		return fmt.Errorf("puerto configurado no válido")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Port() != port {
		return fmt.Errorf("el enlace no corresponde al puerto de esta instalación")
	}
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	job, err := exec.CommandContext(probeCtx, "/bin/launchctl", "print", "system/app.aidex").Output()
	if err != nil {
		return fmt.Errorf("el servicio AIDEX no está registrado o no puede comprobarse")
	}
	pid, err := registeredPID(string(job))
	if err != nil {
		return err
	}
	const executable = "/Library/Application Support/AIDEX/bin/aidex-server"
	if err = trustedExecutable(executable); err != nil {
		return err
	}
	account, err := user.Lookup("_aidex")
	if err != nil {
		return fmt.Errorf("no se pudo comprobar la cuenta del servicio")
	}
	uid, err := strconv.ParseUint(account.Uid, 10, 32)
	if err != nil || uid < 400 || uid >= 500 {
		return fmt.Errorf("la cuenta del servicio no es válida")
	}
	identity, err := nativeProcessIdentity(pid)
	if err != nil {
		return err
	}
	if !matchesServiceProcess(identity, executable, uint32(uid)) {
		return fmt.Errorf("el proceso registrado no corresponde al ejecutable y cuenta de AIDEX; no se modificaron permisos")
	}
	// -Fp también incluye campos f en macOS. -t devuelve exclusivamente PID.
	// No filtrar por PID: hay que detectar también un listener de otro proceso.
	listeners, err := exec.CommandContext(probeCtx, "/usr/sbin/lsof", "-nP", "-iTCP:"+port, "-sTCP:LISTEN", "-t").Output()
	if err != nil || !listenerMatches(string(listeners), pid) {
		return fmt.Errorf("el puerto no pertenece exclusivamente al servicio AIDEX; no se modificaron permisos")
	}
	// Si el servicio reinició o el PID fue reutilizado durante la comprobación,
	// no continuar con permisos basados en una observación anterior.
	job, err = exec.CommandContext(probeCtx, "/bin/launchctl", "print", "system/app.aidex").Output()
	if err != nil {
		return fmt.Errorf("el servicio cambió durante la comprobación; vuelve a intentarlo")
	}
	currentPID, err := registeredPID(string(job))
	if err != nil || currentPID != pid {
		return fmt.Errorf("el servicio reinició; vuelve a intentar la autorización")
	}
	current, err := nativeProcessIdentity(pid)
	if err != nil || current != identity {
		return fmt.Errorf("la identidad del servicio cambió; no se modificaron permisos")
	}
	return nil
}

// No confiar en un binario sustituible por un usuario o por un enlace externo.
func trustedExecutable(path string) error {
	for current := path; current != "/"; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || current == path && !info.Mode().IsRegular() || current != path && !info.IsDir() {
			return fmt.Errorf("no se pudo validar el ejecutable instalado")
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != 0 || info.Mode().Perm()&0022 != 0 {
			return fmt.Errorf("los permisos del ejecutable instalado no son seguros; reinstala AIDEX")
		}
	}
	return nil
}

func main() {
	watchdog := time.AfterFunc(4*time.Minute, func() {
		fmt.Fprintln(os.Stderr, "La autorización superó el tiempo permitido. No se declara éxito; comprueba el acceso antes de reintentar.")
		os.Exit(1)
	})
	defer watchdog.Stop()
	if err := run(); err != nil {
		if errors.Is(err, errTraversalDenied) {
			os.Exit(3)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
