package v1

import (
	"io/ioutil"
	"strings"

	"github.com/tinyzimmer/k3p/pkg/types"
)

var mockArtifacts = []*types.Artifact{
	{
		Type: types.ArtifactBin,
		Name: "k3s",
		Body: ioutil.NopCloser(strings.NewReader("test")),
	},
	{
		Type: types.ArtifactImages,
		Name: "k3s-airgap-images.tar",
		Body: ioutil.NopCloser(strings.NewReader("test")),
	},
	{
		Type: types.ArtifactScript,
		Name: "install.sh",
		Body: ioutil.NopCloser(strings.NewReader("test")),
	},
	{
		Type: types.ArtifactManifest,
		Name: "manifest.yaml",
		Body: ioutil.NopCloser(strings.NewReader("test")),
	},
}

// Mock returns a fake package that can be passed to v1.Load().
func Mock() types.Package {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	writer := &readWriter{workDir: tmpDir, meta: types.NewEmptyMeta()}
	for _, artifact := range mockArtifacts {
		if err := writer.Put(artifact); err != nil {
			panic(err)
		}
	}
	return writer
}
