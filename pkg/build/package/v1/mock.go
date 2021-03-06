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
		Size: 4,
	},
	{
		Type: types.ArtifactImages,
		Name: "k3s-airgap-images.tar",
		Body: ioutil.NopCloser(strings.NewReader("test")),
		Size: 4,
	},
	{
		Type: types.ArtifactScript,
		Name: "install.sh",
		Body: ioutil.NopCloser(strings.NewReader("test")),
		Size: 4,
	},
	{
		Type: types.ArtifactManifest,
		Name: "manifest.yaml",
		Body: ioutil.NopCloser(strings.NewReader("test")),
		Size: 4,
	},
}

// Mock returns a fake package.
func Mock() types.Package {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	writer := New(tmpDir)
	for _, artifact := range mockArtifacts {
		if err := writer.Put(artifact); err != nil {
			panic(err)
		}
	}
	return writer
}
