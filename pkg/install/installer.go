package install

import (
	"io"
	"os"
	"os/exec"
	"path"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

const k3sManifestsDir = "/var/lib/rancher/k3s/server/manifests"
const k3sImagesDir = "/var/lib/rancher/k3s/agent/images"
const k3sScriptdir = "/usr/local/bin/k3p"

// Installer is an interface for laying a package manifest down on a system
// and setting up K3s.
type Installer interface {
	Install(*types.PackageManifest, *Options) error
}

// Options is a placeholder for later options to be used when configuring
// installations.
type Options struct{}

// New returns a new package installer.
func New() Installer { return &installer{} }

type installer struct{}

func (i *installer) Install(manifest *types.PackageManifest, opts *Options) error {

	log.Info("Installing binaries to /usr/local/bin/")
	for _, bin := range manifest.Bins {
		defer bin.Body.Close()
		f, err := os.OpenFile(path.Join("/usr/local/bin", bin.Name), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		defer f.Close()
		log.Debugf("Writing %q to %q", bin.Name, f.Name())
		if _, err := io.Copy(f, bin.Body); err != nil {
			return err
		}
	}

	log.Info("Installing scripts to", k3sScriptdir)
	if err := os.MkdirAll(k3sScriptdir, 0755); err != nil {
		return err
	}
	for _, script := range manifest.Scripts {
		defer script.Body.Close()
		f, err := os.Create(path.Join(k3sScriptdir, script.Name))
		if err != nil {
			return err
		}
		defer f.Close()
		log.Debugf("Writing %q to %q", script.Name, f.Name())
		if _, err := io.Copy(f, script.Body); err != nil {
			return err
		}
	}

	log.Info("Installing images to", k3sImagesDir)
	if err := os.MkdirAll(k3sImagesDir, 0755); err != nil {
		return err
	}
	for _, img := range manifest.Images {
		defer img.Body.Close()
		f, err := os.Create(path.Join(k3sImagesDir, img.Name))
		if err != nil {
			return err
		}
		defer f.Close()
		log.Debugf("Writing %q to %q", img.Name, f.Name())
		if _, err := io.Copy(f, img.Body); err != nil {
			return err
		}
	}

	log.Info("Installing kubernetes manifests to", k3sManifestsDir)
	if err := os.MkdirAll(k3sManifestsDir, 0755); err != nil {
		return err
	}
	for _, mani := range manifest.Manifests {
		defer mani.Body.Close()
		out := path.Join(k3sManifestsDir, mani.Name)
		if err := os.MkdirAll(path.Dir(out), 0755); err != nil {
			return err
		}
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		defer f.Close()
		log.Debugf("Writing %q to %q", mani.Name, f.Name())
		if _, err := io.Copy(f, mani.Body); err != nil {
			return err
		}
	}

	log.Info("Running k3s installation script")
	os.Setenv("INSTALL_K3S_SKIP_DOWNLOAD", "true")
	cmd := exec.Command("/bin/sh", path.Join(k3sScriptdir, "install.sh"))
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	go io.Copy(os.Stdout, outPipe)
	go io.Copy(os.Stderr, errPipe)

	if err := cmd.Run(); err != nil {
		return err
	}

	return cmd.Wait()
}
