package types

// PackageMeta represents metadata included with a package
type PackageMeta struct {
	MetaVersion string `json:"apiVersion"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	K3sVersion  string `json:"k3sVersion"`
	Arch        string `json:"arch"`
}
