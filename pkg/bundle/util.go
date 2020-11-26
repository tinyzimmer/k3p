package bundle

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
)

func getLatestK3sVersion() (string, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	u := fmt.Sprintf("%s/%s", k3sReleasesRootURL, VersionLatest)
	resp, err := client.Get(u)
	if err != nil {
		return "", err
	}
	latestURL := resp.Header.Get("Location")
	return path.Base(latestURL), nil
}

func downloadURLTo(url string, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func calculateSha256Sum(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
