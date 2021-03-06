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
	"github.com/tinyzimmer/k3p/pkg/util"
)

const (
	k3sScriptURL       = "https://get.k3s.io"
	k3sReleasesRootURL = "https://github.com/k3s-io/k3s/releases"
	k3sChannelsRoot    = "https://update.k3s.io/v1-release/channels"
)

func (b *builder) downloadCoreK3sComponents(opts *types.BuildOptions) error {
	log.Info("Fetching checksums...")
	if err := b.downloadK3sChecksums(opts.K3sVersion, opts.Arch); err != nil {
		return err
	}

	log.Info("Fetching k3s install script...")
	if err := b.downloadK3sInstallScript(); err != nil {
		return err
	}

	log.Info("Fetching k3s binary...")
	if err := b.downloadK3sBinary(opts.K3sVersion, opts.Arch); err != nil {
		return err
	}

	if !opts.ExcludeImages {
		log.Info("Fetching k3s airgap images...")
		if err := b.downloadK3sAirgapImages(opts.K3sVersion, opts.Arch); err != nil {
			return err
		}
	} else {
		log.Info("Skipping bundling k3s airgap images with the package")
	}

	log.Info("Validating checksums...")
	if err := b.validateCheckSums(opts); err != nil {
		return err
	}

	return nil
}

func (b *builder) downloadK3sChecksums(version, arch string) error {
	rdr, err := cache.DefaultCache.Get(getDownloadURL(version, getDownloadChecksumsName(arch)))
	if err != nil {
		return err
	}
	artifact, err := util.ArtifactFromReader(types.ArtifactType("misc"), "k3s-sha256sums.txt", rdr)
	if err != nil {
		return err
	}
	return b.writer.Put(artifact)
}

func (b *builder) downloadK3sInstallScript() error {
	rdr, err := cache.DefaultCache.Get(k3sScriptURL)
	if err != nil {
		return err
	}
	artifact, err := util.ArtifactFromReader(types.ArtifactScript, "install.sh", rdr)
	if err != nil {
		return err
	}
	return b.writer.Put(artifact)
}

func (b *builder) downloadK3sAirgapImages(version, arch string) error {
	rdr, err := cache.DefaultCache.Get(getDownloadURL(version, getDownloadAirgapImagesName(arch)))
	if err != nil {
		return err
	}
	artifact, err := util.ArtifactFromReader(types.ArtifactImages, "k3s-airgap-images.tar", rdr)
	if err != nil {
		return err
	}
	return b.writer.Put(artifact)
}

func (b *builder) downloadK3sBinary(version, arch string) error {
	rdr, err := cache.DefaultCache.Get(getDownloadURL(version, getDownloadK3sBinName(arch)))
	if err != nil {
		return err
	}
	artifact, err := util.ArtifactFromReader(types.ArtifactBin, "k3s", rdr)
	if err != nil {
		return err
	}
	return b.writer.Put(artifact)
}

func (b *builder) validateCheckSums(opts *types.BuildOptions) error {
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
		case getDownloadAirgapImagesName(opts.Arch):
			if opts.ExcludeImages {
				imagesValid = true
				continue
			}
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
		case getDownloadK3sBinName(opts.Arch):
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

func getLatestK3sForChannel(channel string) (string, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	u := fmt.Sprintf("%s/%s", k3sChannelsRoot, channel)
	resp, err := client.Get(u)
	if err != nil {
		return "", err
	}
	latestURL := resp.Header.Get("Location")
	return path.Base(latestURL), nil
}

func getDownloadURL(version, component string) string {
	return fmt.Sprintf("%s/download/%s/%s", k3sReleasesRootURL, version, component)
}

func getDownloadChecksumsName(arch string) string {
	return fmt.Sprintf("sha256sum-%s.txt", arch)
}

func getDownloadAirgapImagesName(arch string) string {
	return fmt.Sprintf("k3s-airgap-images-%s.tar", arch)
}

func getDownloadK3sBinName(arch string) string {
	var binaryName string
	switch arch {
	case "amd64":
		binaryName = "k3s"
	case "arm":
		binaryName = "k3s-armhf"
	case "arm64":
		binaryName = "k3s-arm64"
	}
	return binaryName
}
