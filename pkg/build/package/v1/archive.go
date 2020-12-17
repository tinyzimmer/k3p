package v1

import (
	"encoding/base64"
	"io"
	"os"

	"github.com/klauspost/compress/zstd"
	"github.com/tinyzimmer/k3p/pkg/log"
)

// ZstDictionaryB64 is populated at compilation and contains a pre-trained dictionary
// for compressing k3s images.
var ZstDictionaryB64 string

func getZstDict() ([]byte, error) {
	return base64.StdEncoding.DecodeString(ZstDictionaryB64)
}

type archive struct {
	stat os.FileInfo
	f    *os.File
}

// Reader should return a simple io.ReadCloser for the archive.
func (a *archive) Reader() io.ReadCloser { return a.f }

// WriteTo should dump the contents of the archive to the given file.
func (a *archive) WriteTo(path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, a.f)
	return err
}

// Size should return the size of the archive.
func (a *archive) Size() int64 { return a.stat.Size() }

// CompressTo will compress the contents of the archiver to the given file.
// Compression is done using zstandard and a dictionary pre-trained on k3s
// docker images.
func (a *archive) CompressTo(path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	crdr, err := a.CompressReader()
	if err != nil {
		return err
	}
	defer crdr.Close()
	_, err = io.Copy(out, crdr)
	return err
}

// CompressReader should return an io.ReadCloser who's contents are compressed
// with zstandard.
func (a *archive) CompressReader() (io.ReadCloser, error) {
	dictBytes, err := getZstDict()
	if err != nil {
		return nil, err
	}
	r, w := io.Pipe()
	enc, err := zstd.NewWriter(w, zstd.WithEncoderDict(dictBytes))
	if err != nil {
		return nil, err
	}
	go func() {
		defer w.Close()
		defer enc.Close()
		if _, err := io.Copy(enc, a.f); err != nil {
			log.Error(err)
		}
	}()
	return r, nil
}

type zstReadCloser struct{ rdr *zstd.Decoder }

func (z *zstReadCloser) Read(p []byte) (int, error) { return z.rdr.Read(p) }

func (z *zstReadCloser) Close() error {
	z.rdr.Close()
	return nil
}

// Decompress will return a reader that can be used to access the decompressed contents
// of a zst archive.
func Decompress(rdr io.Reader) (io.ReadCloser, error) {
	dictBytes, err := getZstDict()
	if err != nil {
		return nil, err
	}
	out, err := zstd.NewReader(rdr, zstd.WithDecoderDicts(dictBytes))
	if err != nil {
		return nil, err
	}
	// zst.Decoder does not properly implement a ReadCloser
	return &zstReadCloser{out}, nil
}
