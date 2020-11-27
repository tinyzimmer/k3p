package v1

import (
	"archive/tar"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/tinyzimmer/k3p/pkg/types"
)

const (
	// BinDir is the directory for storing binary artifacts inside a package
	BinDir = "bin"
	// ImageDir is the directory for storing container images inside a package
	ImageDir = "images"
	// ScriptsDir is the directory for storing scripts inside a package
	ScriptsDir = "scripts"
	// ManifestDir is the directory for storing manifests inside a package
	ManifestDir = "manifests"
)

// New returns a new v1 bundle writer.
func New(dir string) types.BundleReadWriter { return &readWriter{workDir: dir} }

type readWriter struct{ workDir string }

func (rw *readWriter) Put(artifact *types.Artifact) error {
	defer artifact.Body.Close()
	outPath := path.Join(rw.dirFromType(artifact.Type), artifact.Name)
	if err := os.MkdirAll(path.Dir(outPath), 0755); err != nil {
		return err
	}
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, artifact.Body)
	return err
}

func (rw *readWriter) Get(artifact *types.Artifact) error {
	path := path.Join(rw.dirFromType(artifact.Type), artifact.Name)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	artifact.Body = f
	return nil
}

func (rw *readWriter) ArchiveTo(path string) error {
	defer os.RemoveAll(rw.workDir)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	tw := tar.NewWriter(f)
	defer tw.Close()

	return filepath.Walk(rw.workDir, func(file string, fileInfo os.FileInfo, err error) error {
		// skip the root directory
		if file == rw.workDir {
			return nil
		}

		// generate tar header
		header, err := tar.FileInfoHeader(fileInfo, file)
		if err != nil {
			return err
		}

		// provide a relative name
		header.Name = rw.stripWorkdirPrefix(file)

		// write header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// if a dir continue
		if fileInfo.IsDir() {
			return nil
		}

		// write the file to the tarball
		data, err := os.Open(file)
		if err != nil {
			return err
		}
		_, err = io.Copy(tw, data)
		return err
	})
}

func (rw *readWriter) stripWorkdirPrefix(path string) string {
	return strings.Replace(path, rw.workDir+"/", "", 1)
}

func (rw *readWriter) dirFromType(t types.ArtifactType) string {
	switch t {
	case types.ArtifactBin:
		return path.Join(rw.workDir, BinDir)
	case types.ArtifactImages:
		return path.Join(rw.workDir, ImageDir)
	case types.ArtifactScript:
		return path.Join(rw.workDir, ScriptsDir)
	case types.ArtifactManifest:
		return path.Join(rw.workDir, ManifestDir)
	}
	return rw.workDir // any other artifacts go to the root of the bundle
}