package types

// Installer is an interface for laying a package manifest down on a system
// and setting up K3s.
type Installer interface {
	Install(*PackageManifest, *InstallOptions) error
}

// InstallOptions is a placeholder for later options to be used when configuring
// installations.
type InstallOptions struct{}
