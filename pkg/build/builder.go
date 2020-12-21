package build

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	v1 "github.com/tinyzimmer/k3p/pkg/build/package/v1"
	"github.com/tinyzimmer/k3p/pkg/images"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/parser"
	"github.com/tinyzimmer/k3p/pkg/types"
	"github.com/tinyzimmer/k3p/pkg/util"
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
	writer types.Package
}

func (b *builder) Build(opts *types.BuildOptions) error {
	defer b.writer.Close()

	if opts.Name == "" {
		opts.Name = util.GetRandomName()
		log.Infof("Generated name for package %q\n", opts.Name)
	} else {
		log.Infof("Building package %q\n", opts.Name)
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

	packageMeta := types.PackageMeta{
		MetaVersion: "v1",
		Name:        opts.Name,
		Version:     opts.BuildVersion,
		K3sVersion:  opts.K3sVersion,
		Arch:        opts.Arch,
	}

	if opts.ConfigFile != "" {
		log.Debugf("Reading configuration file at %q\n", opts.ConfigFile)
		conf, err := types.PackageConfigFromFile(opts.ConfigFile)
		if err != nil {
			return err
		}
		packageMeta.PackageConfig = conf
		log.Debugf("Unmarshaled config: %+v\n", *packageMeta.PackageConfig)
	}

	log.Infof("Packaging distribution for version %q using %q architecture\n", opts.K3sVersion, opts.Arch)

	log.Info("Downloading core k3s components")
	// need to implement cache layer
	if err := b.downloadCoreK3sComponents(opts); err != nil {
		return err
	}

	for _, dir := range opts.ManifestDirs {

		parser := parser.NewManifestParser(dir, opts.Excludes, packageMeta.GetPackageConfig())

		log.Infof("Searching %q for kubernetes manifests to include in the archive\n", dir)
		manifests, err := parser.ParseManifests()
		if err != nil {
			return err
		}
		for _, manifest := range manifests {
			if err := b.writer.Put(manifest); err != nil {
				return err
			}
		}

		if !opts.ExcludeImages {
			log.Info("Parsing discovered manifests for container images to download")
			if err := b.bundleImages(opts, parser); err != nil {
				return err
			}
		} else {
			log.Info("Skipping bundling container images with the package")
		}
	}

	if opts.EULAFile != "" {
		log.Infof("Adding EULA from %q\n", opts.EULAFile)
		stat, err := os.Stat(opts.EULAFile)
		if err != nil {
			return err
		}
		f, err := os.Open(opts.EULAFile)
		if err != nil {
			return err
		}
		if err := b.writer.Put(&types.Artifact{
			Type: types.ArtifactEULA,
			Name: types.ManifestEULAFile,
			Body: f,
			Size: stat.Size(),
		}); err != nil {
			return err
		}
	}

	log.Info("Writing package metadata")
	log.Debugf("Appending meta: %+v\n", packageMeta)
	if err := b.writer.PutMeta(&packageMeta); err != nil {
		return err
	}
	log.Debugf("Complete package meta: %+v\n", b.writer.GetMeta())
	log.Debugf("Complete package manifest: %+v\n", *b.writer.GetMeta().GetManifest())
	if cfg := b.writer.GetMeta().GetPackageConfig(); cfg != nil {
		log.Debugf("Complete package config: %+v\n", *cfg)
	}

	log.Info("Finalizing archive")
	archive, err := b.writer.Archive()
	if err != nil {
		return err
	}

	if opts.RunFile {
		return makeRunFile(opts, archive)
	}

	if opts.Compress {
		compName := fmt.Sprintf("%s.zst", opts.Output)
		log.Infof("Writing version %q of %q to %q\n", opts.BuildVersion, opts.Name, compName)
		return archive.CompressTo(compName)
	}

	log.Infof("Writing version %q of %q to %q\n", opts.BuildVersion, opts.Name, opts.Output)
	return archive.WriteTo(opts.Output)
}

func (b *builder) bundleImages(opts *types.BuildOptions, parser types.ManifestParser) error {
	imageNames, err := parser.ParseImages()
	if err != nil {
		return err
	}

	if opts.ImageFile != "" {
		log.Infof("Reading container images from %q\n", opts.ImageFile)
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

	if len(opts.Images) > 0 {
		log.Infof("Got the following images on the command line: %+v\n", opts.Images)
		imageNames = append(imageNames, opts.Images...)
	}

	log.Info("Detected the following images to bundle with the package:", imageNames)

	if opts.CreateRegistry {
		log.Info("Building private image registry to bundle with the package")
		artifacts, err := images.NewImageDownloader().BuildRegistry(imageNames, opts.Arch, opts.PullPolicy)
		if err != nil {
			return err
		}
		for _, artifact := range artifacts {
			if err := b.writer.Put(artifact); err != nil {
				return err
			}
		}
		return nil
	}

	rdr, err := images.NewImageDownloader().SaveImages(imageNames, opts.Arch, opts.PullPolicy)
	if err != nil {
		return err
	}
	log.Info("Adding container images to package")
	images, err := util.ArtifactFromReader(types.ArtifactImages, types.ManifestUserImagesFile, rdr)
	if err != nil {
		return err
	}
	return b.writer.Put(images)
}

var runFilePreSeed = template.Must(template.New("").Parse(`#!/bin/sh

cleanup() { 
	rm -rf {{ .DirName }} 
}

cleanup
trap cleanup EXIT

mkdir -p {{ .DirName }}
tail -n +30 $0 | tar xzf -

if [ -z ${1} ] || [ "${1}" == "install" ] || [ {{ .Backtick }}echo ${1} | cut -c1-1{{ .Backtick }} = "-" ] ; then
	if [ "${1}" == "install" ] ; then shift ; fi
	cmd="./{{ .DirName }}/{{ .K3pBin }} install {{ .DirName }}/{{ .PackageFile }}"
elif [ "${1}" == "inspect" ] ; then
	shift
	cmd="./{{ .DirName }}/{{ .K3pBin }} inspect {{ .DirName }}/{{ .PackageFile }}"
else
	cmd="./{{ .DirName }}/{{ .K3pBin }}"
fi

${cmd} ${@}

exit $?



#payload
`))

var (
	runDirName = ".k3p-run"
	runK3pBin  = "k3p"
)

func tmplSeed(pkgFile string) ([]byte, error) {
	var out bytes.Buffer
	if err := runFilePreSeed.Execute(&out, map[string]string{
		"DirName":     runDirName,
		"K3pBin":      runK3pBin,
		"PackageFile": pkgFile,
		"Backtick":    "`",
	}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func makeRunFile(opts *types.BuildOptions, archive types.Archive) error {
	if strings.HasSuffix(opts.Output, "tar") {
		opts.Output = strings.Replace(opts.Output, ".tar", ".run", 1)
	}

	pkgFile := "package.tar"
	if opts.Compress {
		pkgFile = "package.tar.zst"
	}

	log.Infof("Writing k3p executable and package contents to run file %q\n", opts.Output)

	runFile, err := os.OpenFile(opts.Output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	runFileSeed, err := tmplSeed(pkgFile)
	if err != nil {
		return err
	}
	if _, err := runFile.Write(runFileSeed); err != nil {
		return err
	}

	// wrap the rest of the content in gzip
	gzw := gzip.NewWriter(runFile)

	// Create a new tar writer around the gzip writer
	tw := tar.NewWriter(gzw)

	// get the time
	now := time.Now()

	// downloadURL := fmt.Sprintf("https://github.com/tinyzimmer/k3p/releases/download/%s/k3p_linux_%s", version.K3pVersion, opts.Arch)
	// bin, err := cache.DefaultCache.Get(downloadURL)
	// until i open the repo - the releases can't be downloaded - just use the current executable
	// (which obviously will not always be linux, and it has to be for installations besides docker)
	ex, err := os.Executable()
	if err != nil {
		return err
	}
	bin, err := os.Open(ex)
	if err != nil {
		return err
	}
	binStat, err := os.Stat(ex)
	if err != nil {
		return err
	}

	// Write the k3p binary to the tar ball
	if err := tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     fmt.Sprintf("%s/%s", runDirName, runK3pBin), // must ensure its a linux path separator
		Size:     binStat.Size(),
		Mode:     0755,
		Uid:      0, Gid: 0,
		Uname: "root", Gname: "root",
		ModTime: now, AccessTime: now, ChangeTime: now,
	}); err != nil {
		return err
	}

	if _, err := io.Copy(tw, bin); err != nil {
		return err
	}
	if err := bin.Close(); err != nil {
		return err
	}

	// Write the archive to the tar ball
	rdr := archive.Reader()
	size := archive.Size()
	if opts.Compress {
		// need to compress to a tempfile first
		tmpFile, err := ioutil.TempFile(util.TempDir, "")
		if err != nil {
			return err
		}
		defer os.Remove(tmpFile.Name())
		compressedReader, err := archive.CompressReader()
		if err != nil {
			return err
		}
		if _, err := io.Copy(tmpFile, compressedReader); err != nil {
			return err
		}
		if err := tmpFile.Close(); err != nil {
			return err
		}
		stat, err := os.Stat(tmpFile.Name())
		if err != nil {
			return err
		}
		// Overwrite the size and the reader
		size = stat.Size()
		rdr, err = os.Open(tmpFile.Name())
		if err != nil {
			return err
		}
	}
	if err := tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     fmt.Sprintf("%s/%s", runDirName, pkgFile), // must ensure its a linux path separator
		Size:     size,
		Mode:     0644,
		Uid:      0, Gid: 0,
		Uname: "root", Gname: "root",
		ModTime: now, AccessTime: now, ChangeTime: now,
	}); err != nil {
		return err
	}

	if _, err := io.Copy(tw, rdr); err != nil {
		return err
	}

	// Close the tar writer
	if err := tw.Close(); err != nil {
		return err
	}

	// Close the gzip writer
	if err := gzw.Close(); err != nil {
		return err
	}

	// Close the runfile
	return runFile.Close()
}
