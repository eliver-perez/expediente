package documentformat

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/csv"
	"io"
	"os"
	"unicode/utf16"
	"unicode/utf8"
)

// TextReader accepts UTF-8 and explicitly marked UTF-16. It does not guess a
// single-byte encoding: that would turn arbitrary binary files into valid text.
func TextReader(ctx context.Context, file *os.File, size int64) (io.Reader, string, error) {
	header := make([]byte, min(size, 3))
	if _, err := file.ReadAt(header, 0); err != nil && err != io.EOF {
		return nil, "", err
	}
	offset, encoding := int64(0), "UTF-8"
	var order binary.ByteOrder
	if len(header) >= 2 && header[0] == 0xff && header[1] == 0xfe {
		order, encoding, offset = binary.LittleEndian, "UTF-16LE", 2
	}
	if len(header) >= 2 && header[0] == 0xfe && header[1] == 0xff {
		order, encoding, offset = binary.BigEndian, "UTF-16BE", 2
	}
	if len(header) == 3 && string(header) == "\xef\xbb\xbf" {
		offset = 3
	}
	var reader io.Reader = cancellableReader{ctx, io.NewSectionReader(file, offset, size-offset)}
	if order != nil {
		reader = &utf16Reader{input: bufio.NewReader(reader), order: order}
	}
	return reader, encoding, nil
}

type utf16Reader struct {
	input   *bufio.Reader
	order   binary.ByteOrder
	pending []byte
}

func (r *utf16Reader) word() (uint16, error) {
	var data [2]byte
	_, err := io.ReadFull(r.input, data[:])
	if err == io.ErrUnexpectedEOF {
		return 0, Failure("FILE_TYPE_UNKNOWN")
	}
	return r.order.Uint16(data[:]), err
}
func (r *utf16Reader) Read(output []byte) (int, error) {
	if len(output) == 0 {
		return 0, nil
	}
	if len(r.pending) == 0 {
		first, err := r.word()
		if err != nil {
			return 0, err
		}
		value := rune(first)
		if first >= 0xd800 && first <= 0xdbff {
			second, err := r.word()
			if err == io.EOF {
				return 0, Failure("FILE_TYPE_UNKNOWN")
			}
			if err != nil {
				return 0, err
			}
			if second < 0xdc00 || second > 0xdfff {
				return 0, Failure("FILE_TYPE_UNKNOWN")
			}
			value = utf16.DecodeRune(value, rune(second))
		} else if utf16.IsSurrogate(value) {
			return 0, Failure("FILE_TYPE_UNKNOWN")
		}
		r.pending = utf8.AppendRune(r.pending, value)
	}
	count := copy(output, r.pending)
	r.pending = r.pending[count:]
	return count, nil
}

// CSVStream bounds a quoted record before encoding/csv allocates its fields.
type CSVStream struct {
	parser    *csv.Reader
	budget    *recordReader
	Encoding  string
	Delimiter rune
}

func (s *CSVStream) Read() ([]string, error)       { s.budget.remaining = 8 << 20; return s.parser.Read() }
func (s *CSVStream) FieldPos(field int) (int, int) { return s.parser.FieldPos(field) }
func csvStream(ctx context.Context, file *os.File, size int64, delimiter rune) (*CSVStream, error) {
	reader, encoding, err := TextReader(ctx, file, size)
	if err != nil {
		return nil, err
	}
	budget := &recordReader{reader: reader}
	parser := csv.NewReader(budget)
	parser.Comma = delimiter
	return &CSVStream{parser, budget, encoding, delimiter}, nil
}
func OpenCSV(ctx context.Context, file *os.File, size int64) (*CSVStream, error) {
	for _, delimiter := range []rune{',', ';', '\t', '|'} {
		stream, err := csvStream(ctx, file, size, delimiter)
		if err != nil {
			return nil, err
		}
		rows, columns := 0, 0
		for {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			record, err := stream.Read()
			if err == io.EOF {
				if rows > 0 && columns > 1 {
					return csvStream(ctx, file, size, delimiter)
				}
				break
			}
			if err != nil {
				break
			}
			rows++
			columns = len(record)
		}
	}
	return nil, Failure("FILE_TYPE_MISMATCH")
}
