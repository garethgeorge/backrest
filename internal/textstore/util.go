package textstore

import (
	"bytes"
	"compress/gzip"
	"io"
)

func tailBytes(b []byte, offset int) []byte {
	if offset >= len(b) {
		return nil
	}
	return b[offset:]
}

func headBytes(b []byte, n int) []byte {
	if n >= len(b) {
		return b
	}
	return b[:n]
}

func compress(data []byte) ([]byte, error) {
	gzipBuf := new(bytes.Buffer)
	gzipWriter := gzip.NewWriter(gzipBuf)
	if _, err := gzipWriter.Write(data); err != nil {
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}
	return gzipBuf.Bytes(), nil
}

func decompress(data []byte) ([]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()
	return io.ReadAll(gzipReader)
}
