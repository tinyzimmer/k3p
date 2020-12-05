package build

import (
	"io/ioutil"
	"os"
	"strings"

	v1 "github.com/tinyzimmer/k3p/pkg/build/archive/v1"
	"github.com/tinyzimmer/k3p/pkg/images"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/parser"
	"github.com/tinyzimmer/k3p/pkg/types"
	"github.com/tinyzimmer/k3p/pkg/util"

	"github.com/docker/docker/pkg/namesgenerator"
)

// NewBuilder returns a new Builder for the given K3s version and architecture. If tmpDir
// is empty, the system default is used.
func NewBuilder() (types.Builder, error) {
	// Set up a temporary directory
	tmpDir, err := util.GetTempDir()
	if err != nil {
		return nil, err
	}
	log.Debug("Using temporary build directory:", tmpDir)
	return &builder{writer: v1.New(tmpDir)}, nil
}

// builder implements the Builder interface.
type builder struct {
	// the directory for storing temporary assets during the build
	writer types.BundleReadWriter
}

func (b *builder) Build(opts *types.BuildOptions) error {
	defer b.writer.Close()

	if opts.Name == "" {
		opts.Name = namesgenerator.GetRandomName(0)
	}

	if opts.K3sVersion == types.VersionLatest {
		log.Info("Detecting latest k3s version for channel", opts.K3sChannel)
		latest, err := getLatestK3sForChannel(opts.K3sChannel)
		if err != nil {
			return err
		}
		opts.K3sVersion = latest
		log.Info("Latest k3s version is", opts.K3sVersion)
	}

	log.Infof("Packaging distribution for version %q using %q architecture\n", opts.K3sVersion, opts.Arch)

	log.Info("Downloading core k3s components")
	// need to implement cache layer
	if err := b.downloadCoreK3sComponents(opts.K3sVersion, opts.Arch); err != nil {
		return err
	}

	parser := parser.NewManifestParser(opts.ManifestDir, opts.Excludes, opts.HelmArgs)

	log.Info("Searching for kubernetes manifests to include in the archive")
	manifests, err := parser.ParseManifests()
	if err != nil {
		return err
	}
	for _, manifest := range manifests {
		if err := b.writer.Put(manifest); err != nil {
			return err
		}
	}

	log.Info("Parsing kubernetes manifests for container images to download")
	imageNames, err := parser.ParseImages()
	if err != nil {
		return err
	}

	if opts.ImageFile != "" {
		log.Infof("Reading container images from %q", opts.ImageFile)
		body, err := ioutil.ReadFile(opts.ImageFile)
		if err != nil {
			return err
		}
		for _, img := range strings.Split(string(body), "\n") {
			if img != "" && !strings.HasPrefix(img, "#") {
				imageNames = append(imageNames, img)
			}
		}
	}

	log.Info("Detected the following images to bundle with the package:", imageNames)
	rdr, err := images.NewImageDownloader().PullImages(imageNames, opts.Arch)
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

	if opts.EULAFile != "" {
		log.Infof("Adding EULA from %q\n", opts.EULAFile)
		f, err := os.Open(opts.EULAFile)
		if err != nil {
			return err
		}
		if err := b.writer.Put(&types.Artifact{
			Name: types.ManifestEULAFile,
			Body: f,
		}); err != nil {
			return err
		}
	}

	log.Info("Writing package metadata")
	packageMeta := types.PackageMeta{
		MetaVersion: "v1",
		Name:        opts.Name,
		Version:     opts.BuildVersion,
		K3sVersion:  opts.K3sVersion,
		Arch:        opts.Arch,
	}
	log.Debugf("Package meta: %+v\n", packageMeta)
	if err := b.writer.PutMeta(&packageMeta); err != nil {
		return err
	}

	log.Infof("Archiving version %q of bundle to %q\n", opts.BuildVersion, opts.Output)
	return b.writer.ArchiveTo(opts.Output)
}
