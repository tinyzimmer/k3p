package util

import (
	"crypto/sha256"
	"fmt"
	"io"
)

// CalculateSHA256Sum calculates the sha256sum of the contents of the given reader.
func CalculateSHA256Sum(rdr io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, rdr); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
