package bundle

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/tinyzimmer/k3p/pkg/images"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/parser"
)

const (
	// VersionLatest is a string signaling that the latest version should be retrieved for k3s.
	VersionLatest string = "latest"

	k3sReleasesRootURL string = "https://github.com/rancher/k3s/releases"
)

// Builder is an interface for building application bundles to be distributed to systems.
type Builder interface {
	Setup() error
	Build(*BuildOptions) error
}

// BuildOptions is a struct containing options to pass to the build operation.
type BuildOptions struct {
	ManifestDir string
	Excludes    []string
	Output      string
}

// NewBuilder returns a new Builder for the given K3s version and architecture.
func NewBuilder(version, arch string) Builder {
	return &builder{version: version, arch: arch}
}

// builder implements the Builder interface.
type builder struct {
	// the k3s version to bundle in the package
	version string
	// the architecture to download images and binaries for
	arch string
	// the directory for storing temporary assets during the build
	buildDir string
}

func (b *builder) Setup() error {
	// If using the latest version, fetch the actual semver value
	if b.version == VersionLatest {
		log.Info("Detecting latest k3s version")
		latest, err := getLatestK3sVersion()
		if err != nil {
			return err
		}
		b.version = latest
		log.Info("Latest k3s version is", b.version)
	}

	// Set up a temporary directory
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	b.buildDir = tmpDir
	log.Debug("Using temporary build directory:", b.buildDir)

	// Build out the temp dir structure
	for _, dir := range []string{"bin", "images", "scripts", "manifests"} {
		if err := os.MkdirAll(path.Join(b.buildDir, dir), 0755); err != nil {
			return err
		}
	}

	return nil
}

func (b *builder) Build(opts *BuildOptions) error {
	log.Infof("Packaging distribution for version %q using %q architecture\n", b.version, b.arch)
	defer os.RemoveAll(b.buildDir)

	log.Info("Downloading core k3s components")
	// need to implement cache layer
	if err := b.downloadCoreK3sComponents(); err != nil {
		return err
	}

	log.Info("Parsing kubernetes manifests for container images to download")
	parser := parser.NewImageParser(opts.ManifestDir, opts.Excludes, parser.TypeRaw)

	imageNames, err := parser.Parse()
	if err != nil {
		return err
	}

	log.Info("Detected the following images to bundle with the package:", imageNames)
	downloader := images.NewImageDownloader()
	if err := downloader.PullImages(imageNames); err != nil {
		return err
	}
	if err := downloader.SaveImagesTo(imageNames, path.Join(b.getImagesDir(), "manifest-images.tar")); err != nil {
		return err
	}

	return nil
}

func (b *builder) getDownloadURL(component string) string {
	return fmt.Sprintf("%s/download/%s/%s", k3sReleasesRootURL, b.version, component)
}

func (b *builder) getBinDir() string { return path.Join(b.buildDir, "bin") }

func (b *builder) getScriptDir() string { return path.Join(b.buildDir, "scripts") }

func (b *builder) getImagesDir() string { return path.Join(b.buildDir, "images") }

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
	return downloadURLTo(
		b.getDownloadURL(b.getDownloadChecksumsName()),
		path.Join(b.buildDir, "sha256sum.txt"),
	)
}

func (b *builder) downloadK3sInstallScript() error {
	return downloadURLTo(
		"https://get.k3s.io",
		path.Join(b.getScriptDir(), "install.sh"),
	)
}

func (b *builder) downloadK3sAirgapImages() error {
	return downloadURLTo(
		b.getDownloadURL(b.getDownloadAirgapImagesName()),
		path.Join(b.getImagesDir(), "k3s-airgap-images.tar"),
	)
}

func (b *builder) downloadK3sBinary() error {
	return downloadURLTo(
		b.getDownloadURL(b.getDownloadK3sBinName()),
		path.Join(b.getBinDir(), "k3s"),
	)
}

func (b *builder) validateCheckSums() error {
	var imagesValid, binValid bool

	sumFile, err := os.Open(path.Join(b.buildDir, "sha256sum.txt"))
	if err != nil {
		return err
	}
	defer sumFile.Close()

	scanner := bufio.NewScanner(sumFile)
	for scanner.Scan() {
		text := scanner.Text()
		spl := strings.Fields(text)
		if len(spl) != 2 {
			continue
		}
		shasum, fname := spl[0], spl[1]
		switch fname {
		case b.getDownloadAirgapImagesName():
			localSum, err := calculateSha256Sum(path.Join(b.getImagesDir(), "k3s-airgap-images.tar"))
			if err != nil {
				return err
			}
			if localSum == shasum {
				imagesValid = true
			} else {
				log.Error("Downloaded airgap images sha256sum is invalid")
			}
		case b.getDownloadK3sBinName():
			localSum, err := calculateSha256Sum(path.Join(b.getBinDir(), "k3s"))
			if err != nil {
				return err
			}
			if localSum == shasum {
				binValid = true
			} else {
				log.Error("Downloaded k3s binary sha256sum is invalid")
			}
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return err
	}

	if !imagesValid || !binValid {
		return errors.New("Downloaded files did not match the provided checksums")
	}

	return nil
}
