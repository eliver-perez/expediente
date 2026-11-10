package documentformat

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestUnicodeReaderPreservesCancellation(t *testing.T) {
	for _, contents := range [][]byte{{0xff, 0xfe, 'A', 0}, {0xfe, 0xff, 0, 'A'}} {
		path := filepath.Join(t.TempDir(), "text.txt")
		if err := os.WriteFile(path, contents, 0600); err != nil {
			t.Fatal(err)
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		reader, _, err := TextReader(ctx, file, int64(len(contents)))
		cancel()
		if err == nil {
			_, err = io.ReadAll(reader)
		}
		file.Close()
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled read reported as malformed input: %v", err)
		}
	}
}
