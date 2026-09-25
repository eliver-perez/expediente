package licensing

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

type installationIdentity struct {
	InstallationID string `json:"installation_id"`
	PrivateSeed    string `json:"private_seed"`
	PublicKey      string `json:"public_key"`
	CreatedAt      string `json:"created_at"`
}

func identityPath(stateDirectory string) string {
	return filepath.Join(stateDirectory, "license", "installation.json")
}
func newInstallation() (installationIdentity, ed25519.PrivateKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return installationIdentity{}, nil, err
	}
	identity := installationIdentity{InstallationID: domain.NewID(), PrivateSeed: base64.RawURLEncoding.EncodeToString(privateKey.Seed()), PublicKey: base64.RawURLEncoding.EncodeToString(publicKey), CreatedAt: domain.Timestamp(time.Now())}
	return identity, privateKey, nil
}
func writeIdentity(path string, identity installationIdentity) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	if err = storage.ProtectPrivatePath(path, false); err != nil {
		return err
	}
	if err = json.NewEncoder(file).Encode(identity); err != nil {
		return err
	}
	return file.Sync()
}
func loadInstallation(ctx context.Context, database *storage.Database, stateDirectory, fingerprint string) (Binding, ed25519.PrivateKey, error) {
	binding := Binding{FingerprintVersion: "1", FingerprintHash: fingerprint}
	if _, err := os.Lstat(recoveryPath(stateDirectory)); err == nil || !os.IsNotExist(err) {
		return binding, nil, failure("LICENSE_RECOVERY_PENDING", "Completa la recuperación local pendiente antes de iniciar la licencia.")
	}
	path := identityPath(stateDirectory)
	if err := storage.PreparePrivateDirectory(filepath.Dir(path)); err != nil {
		return binding, nil, err
	}
	var savedID, savedPublic string
	err := database.Reader.QueryRowContext(ctx, "SELECT installation_id,public_key FROM license_installation WHERE singleton=1").Scan(&savedID, &savedPublic)
	if err != nil && err != sql.ErrNoRows {
		return binding, nil, err
	}
	hasAnchor := err == nil
	var identity installationIdentity
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		if hasAnchor {
			return binding, nil, failure("LICENSE_IDENTITY_LOST", "Falta la clave de instalación. Restaura su respaldo o utiliza la recuperación local.")
		}
		var privateKey ed25519.PrivateKey
		identity, privateKey, err = newInstallation()
		if err != nil {
			return binding, nil, err
		}
		if err = writeIdentity(path, identity); err != nil {
			return binding, nil, err
		}
		_ = privateKey
	} else if err != nil {
		return binding, nil, err
	} else {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > 8192 {
			return binding, nil, failure("LICENSE_IDENTITY_INVALID", "La identidad local requiere recuperación.")
		}
		file, err := os.Open(path)
		if err != nil {
			return binding, nil, err
		}
		opened, err := file.Stat()
		if err != nil || !os.SameFile(info, opened) {
			file.Close()
			return binding, nil, failure("LICENSE_IDENTITY_INVALID", "La identidad cambió durante la apertura.")
		}
		contents, err := io.ReadAll(io.LimitReader(file, 8193))
		file.Close()
		if err != nil || len(contents) > 8192 {
			return binding, nil, failure("LICENSE_IDENTITY_INVALID", "La identidad local requiere recuperación.")
		}
		object, err := strictObject(contents)
		if err != nil || !fieldsPresent(object, "installation_id", "private_seed", "public_key", "created_at") || json.Unmarshal(contents, &identity) != nil {
			return binding, nil, failure("LICENSE_IDENTITY_INVALID", "La identidad local requiere recuperación.")
		}
		if err = storage.ProtectPrivatePath(path, false); err != nil {
			return binding, nil, err
		}
	}
	seed, err := decodeBase64(identity.PrivateSeed, ed25519.SeedSize)
	if err != nil || !installationPattern.MatchString(identity.InstallationID) {
		return binding, nil, failure("LICENSE_IDENTITY_INVALID", "La identidad local requiere recuperación.")
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	if base64.RawURLEncoding.EncodeToString(privateKey.Public().(ed25519.PublicKey)) != identity.PublicKey || hasAnchor && (savedID != identity.InstallationID || savedPublic != identity.PublicKey) {
		return binding, nil, failure("LICENSE_IDENTITY_MISMATCH", "La clave local no coincide con esta base de datos.")
	}
	if !hasAnchor {
		err = database.Write(ctx, func(transaction *sql.Tx) error {
			if _, err := transaction.ExecContext(ctx, "INSERT INTO license_installation VALUES(1,?,?,?,?,?)", identity.InstallationID, identity.PublicKey, "1", fingerprint, identity.CreatedAt); err != nil {
				return err
			}
			return audit.Append(ctx, transaction, time.Now(), audit.Event{Type: "license.installation_created", SystemActor: true, Details: map[string]any{"installation_id": identity.InstallationID, "fingerprint_version": "1"}})
		})
		if err != nil {
			return binding, nil, err
		}
	}
	binding.InstallationID = identity.InstallationID
	binding.PublicKey = identity.PublicKey
	return binding, privateKey, nil
}
