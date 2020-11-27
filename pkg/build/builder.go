package build

import (
	"io/ioutil"

	v1 "github.com/tinyzimmer/k3p/pkg/archive/v1"
	"github.com/tinyzimmer/k3p/pkg/images"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/parser"
	"github.com/tinyzimmer/k3p/pkg/types"
)

const (
	// VersionLatest is a string signaling that the latest version should be retrieved for k3s.
	VersionLatest string = "latest"

	k3sReleasesRootURL string = "https://github.com/rancher/k3s/releases"
)

// Builder is an interface for building application bundles to be distributed to systems.
type Builder interface {
	Setup() error
	Build(*Options) error
}

// Options is a struct containing options to pass to the build operation.
type Options struct {
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
	writer types.BundleReadWriter
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
	log.Debug("Using temporary build directory:", tmpDir)

	b.writer = v1.New(tmpDir)
	return nil
}

func (b *builder) Build(opts *Options) error {
	log.Infof("Packaging distribution for version %q using %q architecture\n", b.version, b.arch)

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
	rdr, err := downloader.SaveImages(imageNames)
	if err != nil {
		return err
	}
	if err := b.writer.Put(&types.Artifact{
		Type: types.ArtifactImages,
		Name: "manifest-images.tar",
		Body: rdr,
	}); err != nil {
		return err
	}

	log.Infof("Archiving bundle to %q", opts.Output)
	return b.writer.ArchiveTo(opts.Output)
}
