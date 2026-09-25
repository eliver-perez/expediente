package licensing

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

type recoveryJournal struct {
	Identity    installationIdentity `json:"identity"`
	PreviousID  string               `json:"previous_installation_id"`
	Fingerprint string               `json:"fingerprint_hash"`
	Reason      string               `json:"reason"`
}

func recoveryPath(directory string) string {
	return filepath.Join(directory, "license", "recovery.json")
}

// RecoverIdentity requires the exclusive state lock and a stopped service. A
// durable private journal makes interruptions recoverable by rerunning this CLI.
// It never frees a commercial activation; that requires provider confirmation.
func RecoverIdentity(ctx context.Context, database *storage.Database, directory, reason string) error {
	if strings.TrimSpace(reason) == "" || len(reason) > 1000 {
		return fmt.Errorf("provide a recovery reason of 1 to 1000 characters")
	}
	if err := storage.PreparePrivateDirectory(filepath.Join(directory, "license")); err != nil {
		return err
	}
	journalPath := recoveryPath(directory)
	var journal recoveryJournal
	if info, err := os.Lstat(journalPath); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > 8192 {
			return fmt.Errorf("invalid license recovery journal")
		}
		contents, err := os.ReadFile(journalPath)
		if err != nil {
			return err
		}
		if _, err = strictObject(contents); err != nil || json.Unmarshal(contents, &journal) != nil {
			return fmt.Errorf("invalid license recovery journal")
		}
	} else if !os.IsNotExist(err) {
		return err
	} else {
		identity, _, err := newInstallation()
		if err != nil {
			return err
		}
		fingerprint, err := MachineFingerprint()
		if err != nil {
			return err
		}
		journal = recoveryJournal{Identity: identity, Fingerprint: fingerprint, Reason: reason}
		err = database.Reader.QueryRowContext(ctx, "SELECT installation_id FROM license_installation WHERE singleton=1").Scan(&journal.PreviousID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		contents, err := json.Marshal(journal)
		if err != nil {
			return err
		}
		file, err := os.OpenFile(journalPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		if err = storage.ProtectPrivatePath(journalPath, false); err == nil {
			_, err = file.Write(contents)
		}
		if err == nil {
			err = file.Sync()
		}
		closeErr := file.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	seed, err := decodeBase64(journal.Identity.PrivateSeed, ed25519.SeedSize)
	if err != nil {
		return fmt.Errorf("invalid recovery key")
	}
	if base64.RawURLEncoding.EncodeToString(ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)) != journal.Identity.PublicKey {
		return fmt.Errorf("recovery key mismatch")
	}
	if !installationPattern.MatchString(journal.Identity.InstallationID) || !fingerprintPattern.MatchString(journal.Fingerprint) {
		return fmt.Errorf("invalid recovery identity")
	}
	identityFile := identityPath(directory)
	if info, err := os.Lstat(identityFile); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("remove invalid identity file links using local administration before recovery")
		}
		contents, err := os.ReadFile(identityFile)
		if err != nil {
			return err
		}
		var current installationIdentity
		_ = json.Unmarshal(contents, &current)
		if current != journal.Identity {
			archive := filepath.Join(directory, "license", "installation.previous-"+journal.Identity.InstallationID+".json")
			if _, err = os.Lstat(archive); !os.IsNotExist(err) {
				return fmt.Errorf("recovery archive already exists; preserve both files and inspect locally")
			}
			if err = storage.ProtectPrivatePath(identityFile, false); err != nil {
				return err
			}
			if err = os.Rename(identityFile, archive); err != nil {
				return err
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if _, err := os.Lstat(identityFile); os.IsNotExist(err) {
		if err = writeIdentity(identityFile, journal.Identity); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	err = database.Write(ctx, func(tx *sql.Tx) error {
		var current string
		err := tx.QueryRowContext(ctx, "SELECT installation_id FROM license_installation WHERE singleton=1").Scan(&current)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if current == journal.Identity.InstallationID {
			return nil
		}
		if current != journal.PreviousID {
			return fmt.Errorf("recovery database identity changed")
		}
		if _, err = tx.ExecContext(ctx, "UPDATE license_revision_floors SET deactivated=1"); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE license_state SET current_jws='',deactivated=0,offline_deactivation_pending=0,last_error_code='' WHERE singleton=1"); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE license_operations SET status='failed',error_code='INVALID_PROOF' WHERE status='prepared'"); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO license_installation VALUES(1,?,?,?,?,?) ON CONFLICT(singleton) DO UPDATE SET installation_id=excluded.installation_id,public_key=excluded.public_key,initial_fingerprint_hash=excluded.initial_fingerprint_hash,created_at=excluded.created_at", journal.Identity.InstallationID, journal.Identity.PublicKey, "1", journal.Fingerprint, journal.Identity.CreatedAt); err != nil {
			return err
		}
		return licenseEvent(ctx, tx, domain.Principal{}, "license.identity_recovered", map[string]any{"previous_installation_id": journal.PreviousID, "installation_id": journal.Identity.InstallationID, "reason": journal.Reason, "commercial_transfer_confirmed": false}, domain.RequestMetadata{})
	})
	if err != nil {
		return err
	}
	return os.Remove(journalPath)
}

// Release eligibility is for the H7 installer after it verifies the signed update
// manifest. It does not expire use of a version which is already installed.
func (client *Client) AllowsRelease(ctx context.Context, publishedAt string) error {
	published, err := utcInstant(publishedAt)
	if err != nil {
		return err
	}
	status, err := client.Status(ctx)
	if err != nil {
		return err
	}
	if status.Development {
		return nil
	}
	if status.State == "unactivated" || status.State == "invalid" || status.State == "revoked" {
		return failure("LICENSE_RELEASE_NOT_ENTITLED", "Se requiere una licencia válida para instalar esta versión.")
	}
	cutoff, err := utcInstant(status.EntitledReleaseUntil)
	if err != nil || published.After(cutoff) {
		return failure("LICENSE_RELEASE_NOT_ENTITLED", "Esta versión se publicó después de la fecha incluida en la licencia.")
	}
	return nil
}
