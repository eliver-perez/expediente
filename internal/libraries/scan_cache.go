package libraries

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"time"
)

// Names reserved by Windows, including on SMB shares mounted from other OSes.
// Do not exclude arbitrary hidden directories: they may contain user documents.
func excludedDirectory(name string) bool {
	return strings.EqualFold(name, "$RECYCLE.BIN") || strings.EqualFold(name, "System Volume Information")
}

func (service *Service) observeCached(ctx context.Context, root Root, file *os.File, relative string, maximumBytes int64) (observation, error) {
	info, err := file.Stat()
	if err != nil {
		return observation{}, err
	}
	identity, strong, err := physicalIdentity(file)
	if err != nil {
		return observation{}, err
	}
	if strong && info.Size() <= maximumBytes && time.Since(info.ModTime()) >= 2*time.Second {
		var hash, hashedAt string
		err = service.Database.Reader.QueryRowContext(ctx, `SELECT sha256,hashed_at FROM file_scan_cache WHERE root_id=? AND relative_path=? AND identity_key=? AND size_bytes=? AND modified_ns=?`, root.ID, relative, identity, info.Size(), info.ModTime().UnixNano()).Scan(&hash, &hashedAt)
		if err != nil && err != sql.ErrNoRows {
			return observation{}, err
		}
		verified, _ := time.Parse(time.RFC3339Nano, hashedAt)
		if err == nil && time.Since(verified) >= 0 && time.Since(verified) < 24*time.Hour {
			after, err := file.Stat()
			if err != nil || !os.SameFile(info, after) || info.Size() != after.Size() || !info.ModTime().Equal(after.ModTime()) {
				return observation{}, scanFailure("FILE_UNSTABLE")
			}
			return observation{Identity: identity, Strong: strong, Relative: relative, Hash: hash, Size: info.Size(), Modified: nowModified(info), ModifiedNS: info.ModTime().UnixNano(), Unchanged: true}, nil
		}
	}
	observed, err := observe(ctx, file, relative, maximumBytes)

	return observed, err
}
func nowModified(info os.FileInfo) string {
	return info.ModTime().UTC().Format("2006-01-02T15:04:05.000000Z")
}

func (service *Service) scanPathError(ctx context.Context, root Root, relative string, cause error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	_, err := service.Database.Writer.ExecContext(ctx, `INSERT INTO scan_errors VALUES(?,?,?,?) ON CONFLICT(root_id,relative_path) DO UPDATE SET error_code=excluded.error_code,occurred_at=excluded.occurred_at`, root.ID, relative, failureCode(cause), now())
	return err
}
