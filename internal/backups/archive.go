package backups

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const magic = "AIDEXBK1"
const maxPackage = int64(128 << 30)
const maxExpanded = int64(256 << 30)
const maxEntries = 1000000
const chunkSize = 64 << 10

type Entry struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}
type RootSummary struct {
	ID        string `json:"id"`
	LibraryID string `json:"library_id"`
	Source    string `json:"source"`
	Path      string `json:"path"`
	Status    string `json:"status"`
}
type Manifest struct {
	Product       string           `json:"product"`
	Format        int              `json:"format"`
	Schema        int              `json:"schema"`
	Build         string           `json:"build"`
	Kind          string           `json:"kind"`
	CreatedAt     string           `json:"created_at"`
	Tables        map[string]int64 `json:"tables"`
	Documents     int64            `json:"documents"`
	PhysicalFiles int64            `json:"physical_files"`
	Roots         []RootSummary    `json:"roots"`
	Entries       []Entry          `json:"entries"`
}

func validEntry(name string) bool {
	if name == "" || len(name) > 180 || strings.ContainsAny(name, "\\\x00:") || strings.HasPrefix(name, "/") || path.Clean(name) != name {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func addZipEntry(archive *zip.Writer, name string, src io.Reader) (Entry, error) {
	if !validEntry(name) {
		return Entry{}, fmt.Errorf("invalid archive entry")
	}
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetModTime(time.Unix(0, 0).UTC())
	header.SetMode(0600)
	dst, err := archive.CreateHeader(header)
	if err != nil {
		return Entry{}, err
	}
	hash := sha256.New()
	size, err := io.Copy(dst, io.TeeReader(src, hash))
	if err != nil {
		return Entry{}, err
	}
	return Entry{Name: name, Size: size, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func seal(plainPath, finalPath, password string) error {
	source, err := os.Open(plainPath)
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := os.OpenFile(finalPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	complete := false
	defer func() {
		target.Close()
		if !complete {
			os.Remove(finalPath)
		}
	}()
	if _, err = target.Write([]byte(magic)); err != nil {
		return err
	}
	if password == "" {
		if _, err = target.Write([]byte{0}); err != nil {
			return err
		}
		_, err = io.Copy(target, source)
		if err != nil {
			return err
		}
		err = target.Sync()
		complete = err == nil
		return err
	}
	if _, err = target.Write([]byte{1}); err != nil {
		return err
	}
	salt := make([]byte, 16)
	prefix := make([]byte, 16)
	if _, err = rand.Read(salt); err != nil {
		return err
	}
	if _, err = rand.Read(prefix); err != nil {
		return err
	}
	if _, err = target.Write(salt); err != nil {
		return err
	}
	if _, err = target.Write(prefix); err != nil {
		return err
	}
	key := argon2.IDKey([]byte(password), salt, 3, 64*1024, 1, 32)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return err
	}
	block := make([]byte, chunkSize)
	nonce := make([]byte, 24)
	copy(nonce, prefix)
	for seq := uint64(0); ; seq++ {
		n, readErr := io.ReadFull(source, block)
		final := readErr == io.EOF || readErr == io.ErrUnexpectedEOF
		if readErr != nil && !final {
			return readErr
		}
		binary.BigEndian.PutUint64(nonce[16:], seq)
		ad := make([]byte, 9)
		binary.BigEndian.PutUint64(ad[:8], seq)
		if final {
			ad[8] = 1
		}
		ciphertext := aead.Seal(nil, nonce, block[:n], ad)
		var length [4]byte
		binary.BigEndian.PutUint32(length[:], uint32(len(ciphertext)))
		if _, err = target.Write(length[:]); err != nil {
			return err
		}
		if _, err = target.Write(ciphertext); err != nil {
			return err
		}
		if final {
			break
		}
	}
	err = target.Sync()
	complete = err == nil
	return err
}

// openPackage decrypts only into a private scratch file. The caller removes it.
func openPackage(sourcePath, scratchPath, password string) (string, error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return "", err
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil {
		return "", err
	}
	if info.Size() > maxPackage || info.Size() < 9 {
		return "", fmt.Errorf("invalid package size")
	}
	header := make([]byte, 9)
	if _, err = io.ReadFull(source, header); err != nil {
		return "", err
	}
	if string(header[:8]) != magic {
		return "", fmt.Errorf("unrecognized backup format")
	}
	if header[8] == 0 {
		if password != "" {
			return "", fmt.Errorf("package is not encrypted")
		}
		return sourcePath, nil
	}
	if header[8] != 1 {
		return "", fmt.Errorf("unsupported encryption format")
	}
	if password == "" {
		return "", fmt.Errorf("backup password required")
	}
	salt := make([]byte, 16)
	prefix := make([]byte, 16)
	if _, err = io.ReadFull(source, salt); err != nil {
		return "", err
	}
	if _, err = io.ReadFull(source, prefix); err != nil {
		return "", err
	}
	target, err := os.OpenFile(scratchPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	complete := false
	defer func() {
		target.Close()
		if !complete {
			os.Remove(scratchPath)
		}
	}()
	key := argon2.IDKey([]byte(password), salt, 3, 64*1024, 1, 32)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, 24)
	copy(nonce, prefix)
	var total int64
	for seq := uint64(0); ; seq++ {
		var raw [4]byte
		if _, err = io.ReadFull(source, raw[:]); err != nil {
			return "", fmt.Errorf("truncated encrypted package: %w", err)
		}
		length := binary.BigEndian.Uint32(raw[:])
		if length < 16 || length > chunkSize+16 {
			return "", fmt.Errorf("invalid encrypted block")
		}
		ciphertext := make([]byte, length)
		if _, err = io.ReadFull(source, ciphertext); err != nil {
			return "", err
		}
		final := length < chunkSize+16
		binary.BigEndian.PutUint64(nonce[16:], seq)
		ad := make([]byte, 9)
		binary.BigEndian.PutUint64(ad[:8], seq)
		if final {
			ad[8] = 1
		}
		plain, openErr := aead.Open(nil, nonce, ciphertext, ad)
		if openErr != nil {
			return "", fmt.Errorf("invalid password or damaged backup")
		}
		total += int64(len(plain))
		if total > maxExpanded {
			return "", fmt.Errorf("package exceeds resource limit")
		}
		if _, err = target.Write(plain); err != nil {
			return "", err
		}
		if final {
			var rest [1]byte
			n, e := source.Read(rest[:])
			if n != 0 || !errors.Is(e, io.EOF) {
				return "", fmt.Errorf("trailing encrypted data")
			}
			break
		}
	}
	if err = target.Sync(); err != nil {
		return "", err
	}
	complete = true
	return scratchPath, nil
}

type Package struct {
	Manifest Manifest
	Files    map[string]*zip.File
	handle   *os.File
}

func (p *Package) Close() error { return p.handle.Close() }
func inspectZip(pathname string) (*Package, error) {
	info, err := os.Lstat(pathname)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("package is not a regular file")
	}
	if info.Size() > maxPackage {
		return nil, fmt.Errorf("package exceeds resource limit")
	}
	// Skip the unencrypted AIDEX header. ZIP offsets are relative to this section.
	handle, err := os.Open(pathname)
	if err != nil {
		return nil, err
	}
	valid := false
	defer func() {
		if !valid {
			handle.Close()
		}
	}()
	header := make([]byte, 9)
	if _, err = io.ReadFull(handle, header); err != nil {
		return nil, err
	}
	offset := int64(0)
	if string(header[:8]) == magic && header[8] == 0 {
		offset = 9
	}
	if info.Size()-offset < 22 {
		return nil, fmt.Errorf("truncated ZIP directory")
	}
	var ending [22]byte
	if _, err = handle.ReadAt(ending[:], info.Size()-22); err != nil {
		return nil, err
	}
	if !bytes.Equal(ending[:4], []byte{'P', 'K', 5, 6}) || binary.LittleEndian.Uint16(ending[20:22]) != 0 {
		return nil, fmt.Errorf("ZIP has trailing data or unsupported comment")
	}
	reader, err := zip.NewReader(io.NewSectionReader(handle, offset, info.Size()-offset), info.Size()-offset)
	if err != nil {
		return nil, err
	}
	if len(reader.File) > maxEntries {
		return nil, fmt.Errorf("too many archive entries")
	}
	files := map[string]*zip.File{}
	var expanded int64
	for _, file := range reader.File {
		if !validEntry(file.Name) || !file.Mode().IsRegular() || file.FileInfo().IsDir() || file.UncompressedSize64 > uint64(maxExpanded) {
			return nil, fmt.Errorf("unsafe archive entry")
		}
		if _, found := files[file.Name]; found {
			return nil, fmt.Errorf("duplicate archive entry")
		}
		if file.Name != "manifest.json" && !strings.HasPrefix(file.Name, "tables/") && !strings.HasPrefix(file.Name, "files/") && !strings.HasPrefix(file.Name, "uploads/") {
			return nil, fmt.Errorf("unauthorized archive entry")
		}
		expanded += int64(file.UncompressedSize64)
		if expanded > maxExpanded {
			return nil, fmt.Errorf("expanded package exceeds resource limit")
		}
		files[file.Name] = file
	}
	manifestFile := files["manifest.json"]
	if manifestFile == nil || manifestFile.UncompressedSize64 > 4<<20 {
		return nil, fmt.Errorf("manifest missing or too large")
	}
	stream, err := manifestFile.Open()
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(stream, 4<<20+1))
	stream.Close()
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err = json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	if manifest.Product != "AIDEX" || manifest.Format != 1 || manifest.Schema != 15 || (manifest.Kind != "basic" && manifest.Kind != "full") {
		return nil, fmt.Errorf("incompatible backup version")
	}
	if len(manifest.Entries) != len(files)-1 {
		return nil, fmt.Errorf("manifest inventory mismatch")
	}
	seen := map[string]bool{}
	for _, expected := range manifest.Entries {
		file := files[expected.Name]
		if file == nil || seen[expected.Name] || expected.Size < 0 || uint64(expected.Size) != file.UncompressedSize64 {
			return nil, fmt.Errorf("manifest entry mismatch")
		}
		seen[expected.Name] = true
		h := sha256.New()
		body, err := file.Open()
		if err != nil {
			return nil, err
		}
		size, err := io.Copy(h, io.LimitReader(body, expected.Size+1))
		body.Close()
		if err != nil || size != expected.Size || hex.EncodeToString(h.Sum(nil)) != expected.SHA256 {
			return nil, fmt.Errorf("backup integrity failure: %s", expected.Name)
		}
	}
	for name := range files {
		if name != "manifest.json" && !seen[name] {
			return nil, fmt.Errorf("unexpected archive entry")
		}
	}
	valid = true
	return &Package{Manifest: manifest, Files: files, handle: handle}, nil
}

func encodeManifest(manifest Manifest) (*bytes.Reader, error) {
	data, err := json.Marshal(manifest)
	return bytes.NewReader(data), err
}
