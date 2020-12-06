package v1

import (
	"io"
	"os"
)

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
