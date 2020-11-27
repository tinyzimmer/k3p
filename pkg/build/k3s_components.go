package build

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/tinyzimmer/k3p/pkg/cache"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

const k3sScriptURL = "https://get.k3s.io"

func (b *builder) getDownloadURL(component string) string {
	return fmt.Sprintf("%s/download/%s/%s", k3sReleasesRootURL, b.version, component)
}

func (b *builder) getDownloadChecksumsName() string {
	return fmt.Sprintf("sha256sum-%s.txt", b.arch)
}

func (b *builder) getDownloadAirgapImagesName() string {
	return fmt.Sprintf("k3s-airgap-images-%s.tar", b.arch)
}

func (b *builder) getDownloadK3sBinName() string {
	var binaryName string
	switch b.arch {
	case "amd64":
		binaryName = "k3s"
	case "arm":
		binaryName = "k3s-armhf"
	case "arm64":
		binaryName = "k3s-arm64"
	}
	return binaryName
}

func (b *builder) downloadCoreK3sComponents() error {
	log.Info("Fetching checksums...")
	if err := b.downloadK3sChecksums(); err != nil {
		return err
	}

	log.Info("Fetching k3s install script...")
	if err := b.downloadK3sInstallScript(); err != nil {
		return err
	}

	log.Info("Fetching k3s binary...")
	if err := b.downloadK3sBinary(); err != nil {
		return err
	}

	log.Info("Fetching k3s airgap images...")
	if err := b.downloadK3sAirgapImages(); err != nil {
		return err
	}

	log.Info("Validating checksums...")
	if err := b.validateCheckSums(); err != nil {
		return err
	}

	return nil
}

func (b *builder) downloadK3sChecksums() error {
	rdr, err := cache.DefaultCache.Get(b.getDownloadURL(b.getDownloadChecksumsName()))
	if err != nil {
		return err
	}
	return b.writer.Put(&types.Artifact{
		Name: "k3s-sha256sums.txt",
		Body: rdr,
	})
}

func (b *builder) downloadK3sInstallScript() error {
	rdr, err := cache.DefaultCache.Get(k3sScriptURL)
	if err != nil {
		return err
	}
	return b.writer.Put(&types.Artifact{
		Type: types.ArtifactScript,
		Name: "install.sh",
		Body: rdr,
	})
}

func (b *builder) downloadK3sAirgapImages() error {
	rdr, err := cache.DefaultCache.Get(b.getDownloadURL(b.getDownloadAirgapImagesName()))
	if err != nil {
		return err
	}
	return b.writer.Put(&types.Artifact{
		Type: types.ArtifactImages,
		Name: "k3s-airgap-images.tar",
		Body: rdr,
	})
}

func (b *builder) downloadK3sBinary() error {
	rdr, err := cache.DefaultCache.Get(b.getDownloadURL(b.getDownloadK3sBinName()))
	if err != nil {
		return err
	}
	return b.writer.Put(&types.Artifact{
		Type: types.ArtifactBin,
		Name: "k3s",
		Body: rdr,
	})
}

func (b *builder) validateCheckSums() error {
	// Queue up extra check to make sure we visited each
	var binValid, imagesValid bool

	// retrieve the downloaded checksums from the bundle
	checksums := &types.Artifact{Name: "k3s-sha256sums.txt"}
	if err := b.writer.Get(checksums); err != nil {
		return err
	}
	defer checksums.Body.Close()

	// scan the file for the image and binary checksums
	scanner := bufio.NewScanner(checksums.Body)
	for scanner.Scan() {

		text := scanner.Text()

		// file is structured as "<checksum> <remote filename>"
		spl := strings.Fields(text)
		if len(spl) != 2 {
			// blank line or a comment
			continue
		}
		shasum, fname := spl[0], spl[1]

		// verify the checksums
		switch fname {
		case b.getDownloadAirgapImagesName():
			images := &types.Artifact{
				Type: types.ArtifactImages,
				Name: "k3s-airgap-images.tar",
			}
			if err := b.writer.Get(images); err != nil {
				return err
			}
			defer images.Body.Close()
			if err := images.Verify(shasum); err != nil {
				return err
			}
			imagesValid = true
		case b.getDownloadK3sBinName():
			k3sbin := &types.Artifact{
				Type: types.ArtifactBin,
				Name: "k3s",
			}
			if err := b.writer.Get(k3sbin); err != nil {
				return err
			}
			defer k3sbin.Body.Close()
			if err := k3sbin.Verify(shasum); err != nil {
				return err
			}
			binValid = true
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return err
	}

	if !binValid || !imagesValid {
		return errors.New("A checksum wasn't present for one of the k3s binary or images")
	}

	return nil
}

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
