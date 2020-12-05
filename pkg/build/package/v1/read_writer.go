package v1

import (
	"archive/tar"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
	"github.com/tinyzimmer/k3p/pkg/util"
)

const (
	// binDir is the directory for storing binary artifacts inside a package
	binDir = "bin"
	// imageDir is the directory for storing container images inside a package
	imageDir = "images"
	// scriptsDir is the directory for storing scripts inside a package
	scriptsDir = "scripts"
	// manifestDir is the directory for storing manifests inside a package
	manifestDir = "manifests"
)

// New returns a new v1 package writer.
func New(dir string) types.Package {
	return &readWriter{
		workDir: dir,
		meta:    types.NewEmptyMeta(),
	}
}

// Load will load the bundle from the given tar archive. TmpDir is the directory to temporarily
// unarchive the contents to. If it is blank, the system default is used.
func Load(rdr io.ReadCloser) (types.Package, error) {
	defer rdr.Close()
	// Create a work directory for the readwriter
	workdir, err := util.GetTempDir()
	if err != nil {
		return nil, err
	}
	log.Debugf("Using temp dir: %q\n", workdir)

	// Extract the contents of the archive into the workdir
	tarReader := tar.NewReader(rdr)
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

	// Load the package meta
	rawMeta, err := ioutil.ReadFile(path.Join(workdir, types.ManifestMetaFile))
	if err != nil {
		return nil, err
	}
	meta := types.NewEmptyMeta()
	if err := json.Unmarshal(rawMeta, meta); err != nil {
		return nil, err
	}

	// Return a readwriter using the extracted directory
	return &readWriter{workDir: workdir, meta: meta}, nil
}

type readWriter struct {
	workDir string
	meta    *types.PackageMeta
}

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
	if _, err = io.Copy(out, artifact.Body); err != nil {
		return err
	}
	rw.appendMeta(artifact.Type, outPath)
	return nil
}

func (rw *readWriter) PutMeta(meta *types.PackageMeta) error {
	out, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return json.Unmarshal(out, rw.meta)
}

func (rw *readWriter) GetMeta() *types.PackageMeta { return rw.meta }

func (rw *readWriter) Get(artifact *types.Artifact) error {
	var filePath string

	// this is a terrible hack
	if rw.hasDirPrefix(artifact.Type, artifact.Name) {
		filePath = path.Join(rw.workDir, artifact.Name)
		artifact.Name = rw.stripLocalDirPrefix(artifact.Type, artifact.Name)
	} else {
		filePath = path.Join(rw.dirFromType(artifact.Type), artifact.Name)
	}

	stat, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	artifact.Body = f
	artifact.Size = stat.Size()
	return nil
}

func (rw *readWriter) ArchiveTo(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return rw.archiveToWriter(f)
}

func (rw *readWriter) Close() error { return os.RemoveAll(rw.workDir) }

func (rw *readWriter) archiveToWriter(w io.Writer) error {
	// Write the metadata file
	meta, err := json.MarshalIndent(rw.GetMeta(), "", "  ")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(path.Join(rw.workDir, types.ManifestMetaFile), meta, 0644); err != nil {
		return err
	}

	tw := tar.NewWriter(w)
	defer tw.Close()

	return filepath.Walk(rw.workDir, func(file string, fileInfo os.FileInfo, lastErr error) error {
		if lastErr != nil {
			return lastErr
		}

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

func (rw *readWriter) appendMeta(t types.ArtifactType, fullLocalPath string) {
	strippedPath := rw.stripWorkdirPrefix(fullLocalPath)
	switch t {
	case types.ArtifactBin:
		rw.meta.Manifest.Bins = append(rw.meta.Manifest.Bins, strippedPath)
	case types.ArtifactImages:
		rw.meta.Manifest.Images = append(rw.meta.Manifest.Images, strippedPath)
	case types.ArtifactScript:
		rw.meta.Manifest.Scripts = append(rw.meta.Manifest.Scripts, strippedPath)
	case types.ArtifactManifest:
		rw.meta.Manifest.K8sManifests = append(rw.meta.Manifest.K8sManifests, strippedPath)
	case types.ArtifactEULA:
		rw.meta.Manifest.EULA = strippedPath
	}
}

func (rw *readWriter) hasDirPrefix(t types.ArtifactType, name string) bool {
	switch t {
	case types.ArtifactBin:
		return strings.HasPrefix(name, binDir)
	case types.ArtifactImages:
		return strings.HasPrefix(name, imageDir)
	case types.ArtifactScript:
		return strings.HasPrefix(name, scriptsDir)
	case types.ArtifactManifest:
		return strings.HasPrefix(name, manifestDir)
	}
	return false
}

func (rw *readWriter) stripLocalDirPrefix(t types.ArtifactType, name string) string {
	switch t {
	case types.ArtifactBin:
		return strings.TrimPrefix(name, binDir+"/")
	case types.ArtifactImages:
		return strings.TrimPrefix(name, imageDir+"/")
	case types.ArtifactScript:
		return strings.TrimPrefix(name, scriptsDir+"/")
	case types.ArtifactManifest:
		return strings.TrimPrefix(name, manifestDir+"/")
	}
	return name
}

func (rw *readWriter) dirFromType(t types.ArtifactType) string {
	switch t {
	case types.ArtifactBin:
		return path.Join(rw.workDir, binDir)
	case types.ArtifactImages:
		return path.Join(rw.workDir, imageDir)
	case types.ArtifactScript:
		return path.Join(rw.workDir, scriptsDir)
	case types.ArtifactManifest:
		return path.Join(rw.workDir, manifestDir)
	}
	return rw.workDir // any other artifacts go to the root of the bundle
}
