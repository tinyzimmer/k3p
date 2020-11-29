package v1

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/tinyzimmer/k3p/pkg/log"
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

// Load will load the bundle from the given tar archive. TmpDir is the directory to temporarily
// unarchive the contents to. If it is blank, the system default is used.
func Load(tarPath string, tmpDir string) (types.BundleReadWriter, error) {
	log.Infof("Extracting %q", tarPath)
	// Open the archive
	f, err := os.Open(tarPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Create a work directory for the readwriter
	workdir, err := ioutil.TempDir(tmpDir, "")
	if err != nil {
		return nil, err
	}
	log.Debugf("Using temp dir: %q", workdir)

	// Extract the contents of the archive into the workdir
	tarReader := tar.NewReader(f)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(path.Join(workdir, header.Name), 0755); err != nil {
				return nil, err
			}
		case tar.TypeReg:
			out, err := os.Create(path.Join(workdir, header.Name))
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(out, tarReader); err != nil {
				return nil, err
			}
			out.Close()
		}
	}

	// Return a readwriter using the extracted directory
	return &readWriter{workDir: workdir}, nil
}

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

func (rw *readWriter) GetManifest() (*types.PackageManifest, error) {
	manifest := types.NewPackageManifest()
	return manifest, filepath.Walk(rw.workDir, func(file string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fileInfo.IsDir() {
			return nil
		}

		parts := strings.Split(rw.stripWorkdirPrefix(file), string(os.PathSeparator))
		if len(parts) < 2 { // we are not interested in files at the root of the tarball, though probably will be in the future
			return nil
		}

		f, err := os.Open(file)
		if err != nil {
			return err
		}

		rootDir := parts[0]
		artifact := &types.Artifact{
			Name: strings.Join(parts[1:], string(os.PathSeparator)),
			Size: fileInfo.Size(),
			Body: f,
		}

		switch rootDir {
		case BinDir:
			artifact.Type = types.ArtifactBin
			manifest.Bins = append(manifest.Bins, artifact)
		case ImageDir:
			artifact.Type = types.ArtifactImages
			manifest.Images = append(manifest.Images, artifact)
		case ScriptsDir:
			artifact.Type = types.ArtifactScript
			manifest.Scripts = append(manifest.Scripts, artifact)
		case ManifestDir:
			artifact.Type = types.ArtifactManifest
			manifest.Manifests = append(manifest.Manifests, artifact)
		}

		return nil
	})
}

func (rw *readWriter) Get(artifact *types.Artifact) error {
	path := path.Join(rw.dirFromType(artifact.Type), artifact.Name)
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	artifact.Body = f
	artifact.Size = stat.Size()
	return nil
}

func (rw *readWriter) Close() error { return os.RemoveAll(rw.workDir) }

func (rw *readWriter) ArchiveTo(path string) error {
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
