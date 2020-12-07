package v1

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

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
	// the tar file we use inside the workdir
	tarFile = "package.tar"
)

// New returns a new v1 package writer.
func New(dir string) types.Package {
	return &readWriter{
		workDir: dir,
		meta:    types.NewEmptyMeta(),
	}
}

// Load loads the given readcloser into a Package interface.
func Load(rdr io.ReadCloser) (types.Package, error) {
	defer rdr.Close()
	workDir, err := util.GetTempDir()
	if err != nil {
		return nil, err
	}
	pkg := &readWriter{workDir: workDir}
	f, err := os.Create(pkg.tarFile())
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := io.Copy(f, rdr); err != nil {
		return nil, err
	}

	artifact := &types.Artifact{Name: types.ManifestMetaFile}
	if err := pkg.Get(artifact); err != nil {
		return nil, err
	}
	rawMeta, err := ioutil.ReadAll(artifact.Body)
	if err != nil {
		return nil, err
	}
	var meta types.PackageMeta
	if err := json.Unmarshal(rawMeta, &meta); err != nil {
		return nil, err
	}
	pkg.meta = &meta
	return pkg, nil
}

type readWriter struct {
	workDir string
	meta    *types.PackageMeta
}

func (rw *readWriter) tarFile() string {
	return path.Join(rw.workDir, tarFile)
}

func (rw *readWriter) getTarWriter() (*tar.Writer, error) {
	if fileExists(rw.tarFile()) {
		f, err := os.OpenFile(rw.tarFile(), os.O_RDWR, os.ModePerm)
		if err != nil {
			return nil, err
		}
		if _, err := f.Seek(-1024, os.SEEK_END); err != nil {
			return nil, err
		}
		return tar.NewWriter(f), nil
	}
	f, err := os.Create(rw.tarFile())
	if err != nil {
		return nil, err
	}
	return tar.NewWriter(f), nil
}

func (rw *readWriter) Put(artifact *types.Artifact) error {
	defer artifact.Body.Close()
	tarWriter, err := rw.getTarWriter()
	if err != nil {
		return err
	}
	defer tarWriter.Close()
	header := genArtifactHeader(artifact)
	log.Debug("Generated header for artifact", header)
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	_, err = io.Copy(tarWriter, artifact.Body)
	rw.appendMeta(artifact.Type, header.Name)
	return err
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
	searchName := artifact.Name
	if !rw.hasDirPrefix(artifact) {
		searchName = path.Join(dirFromType(artifact.Type), artifact.Name)
	}
	f, err := os.Open(rw.tarFile())
	if err != nil {
		return err
	}
	rdr := tar.NewReader(f)
	for {
		header, err := rdr.Next()
		if err != nil {
			if err != io.EOF {
				return err
			}
			// reached end of file and did not find artifact
			return fmt.Errorf("%s artifact %q not found", artifact.Type, artifact.Name)
		}
		// Only evaluate files
		if header.Typeflag != tar.TypeReg {
			continue
		}
		// Check if this is the right artifact
		if header.Name != searchName {
			continue
		}
		// We have the right artifact - populate the provided object and return
		artifact.Body = ioutil.NopCloser(rdr)
		artifact.Size = header.Size
		// this produces a race condition, but should really make sure file is properly closed
		// and not rely on waiting for the program to exit
		// runtime.SetFinalizer(artifact, func(_ *types.Artifact) { f.Close() })
		return nil
	}
}

func (rw *readWriter) Archive() (types.Archive, error) {
	rawMeta, err := json.MarshalIndent(rw.meta, "", "  ")
	if err != nil {
		return nil, err
	}
	artifact := &types.Artifact{
		Name: types.ManifestMetaFile,
		Body: ioutil.NopCloser(bytes.NewReader(rawMeta)),
		Size: int64(len(rawMeta)),
	}
	if err := rw.Put(artifact); err != nil {
		return nil, err
	}
	stat, err := os.Stat(rw.tarFile())
	if err != nil {
		return nil, err
	}
	f, err := os.Open(rw.tarFile())
	if err != nil {
		return nil, err
	}
	out := &archive{stat: stat, f: f}
	runtime.SetFinalizer(out, func(_ *archive) { f.Close() })
	return out, nil
}

func (rw *readWriter) Close() error {
	return os.RemoveAll(rw.workDir)
}

func (rw *readWriter) appendMeta(t types.ArtifactType, tarPath string) {
	switch t {
	case types.ArtifactBin:
		rw.meta.Manifest.Bins = append(rw.meta.Manifest.Bins, tarPath)
	case types.ArtifactImages:
		rw.meta.Manifest.Images = append(rw.meta.Manifest.Images, tarPath)
	case types.ArtifactScript:
		rw.meta.Manifest.Scripts = append(rw.meta.Manifest.Scripts, tarPath)
	case types.ArtifactManifest:
		rw.meta.Manifest.K8sManifests = append(rw.meta.Manifest.K8sManifests, tarPath)
	case types.ArtifactEULA:
		rw.meta.Manifest.EULA = tarPath
	}
}

func (rw *readWriter) hasDirPrefix(artifact *types.Artifact) bool {
	switch artifact.Type {
	case types.ArtifactBin:
		return strings.HasPrefix(artifact.Name, binDir)
	case types.ArtifactImages:
		return strings.HasPrefix(artifact.Name, imageDir)
	case types.ArtifactScript:
		return strings.HasPrefix(artifact.Name, scriptsDir)
	case types.ArtifactManifest:
		return strings.HasPrefix(artifact.Name, manifestDir)
	}
	return false
}

func dirFromType(t types.ArtifactType) string {
	switch t {
	case types.ArtifactBin:
		return binDir
	case types.ArtifactImages:
		return imageDir
	case types.ArtifactScript:
		return scriptsDir
	case types.ArtifactManifest:
		return manifestDir
	}
	return ""
}

func fileExists(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !stat.IsDir()
}

func genArtifactHeader(artifact *types.Artifact) *tar.Header {
	uid := 0
	uname := "root"
	if u, err := user.Current(); err == nil {
		if uidInt, err := strconv.Atoi(u.Uid); err == nil {
			uid = uidInt
			uname = u.Username
		}
	}
	now := time.Now()
	return &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     path.Join(dirFromType(artifact.Type), artifact.Name),
		Size:     artifact.Size,
		Mode:     0644,
		Uid:      uid, Gid: uid,
		Uname: uname, Gname: uname,
		ModTime: now, AccessTime: now, ChangeTime: now,
	}
}
