package install

import (
	"io"
	"os"
	"os/exec"
	"path"
	"time"

	v1 "github.com/tinyzimmer/k3p/pkg/build/archive/v1"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

const k3sManifestsDir = "/var/lib/rancher/k3s/server/manifests"
const k3sImagesDir = "/var/lib/rancher/k3s/agent/images"
const k3sScriptdir = "/usr/local/bin/k3p"

// New returns a new package installer.
func New() types.Installer { return &installer{} }

type installer struct{}

func (i *installer) Install(opts *types.InstallOptions) error {
	pkg, err := v1.Load(opts.TarPath)
	if err != nil {
		return err
	}
	defer pkg.Close()

	// check package for a EULA
	eula := &types.Artifact{Name: "EULA.txt"}
	if err := pkg.Get(eula); err == nil {
		// File was found
		if !opts.AcceptEULA {
			pager := os.Getenv("PAGER")
			if pager == "" {
				pager = "less"
			}
			cmd := exec.Command(pager)
			cmd.Stdin = eula.Body
			cmd.Stdout = os.Stdout
			if err := cmd.Run(); err != nil {
				return err
			}
			time.Sleep(time.Second)
		}
	} else if !os.IsNotExist(err) {
		// Error other than file not found
		return err
	}

	manifest, err := pkg.GetManifest()
	if err != nil {
		return err
	}

	if err := i.installManifest(manifest); err != nil {
		return err
	}

	os.Setenv("INSTALL_K3S_SKIP_DOWNLOAD", "true")
	if opts.NodeName != "" {
		log.Info("Using node name:", opts.NodeName)
		os.Setenv("K3S_NODE_NAME", opts.NodeName)
	}
	if opts.ServerURL != "" && opts.NodeToken != "" {
		log.Info("Joining server at:", opts.ServerURL)
		os.Setenv("K3S_URL", opts.ServerURL)
		os.Setenv("K3S_TOKEN", opts.NodeToken)
	}
	if opts.ResolvConf != "" {
		log.Info("Using custom resolv-conf at:", opts.ResolvConf)
		os.Setenv("K3S_RESOLV_CONF", opts.ResolvConf)
	}
	if opts.KubeconfigMode != "" {
		log.Info("Setting admin kubeconfig mode to", opts.KubeconfigMode)
		os.Setenv("K3S_KUBECONFIG_MODE", opts.KubeconfigMode)
	}
	if opts.K3sExecArgs != "" {
		log.Infof("Applying extra k3s arguments: %q", opts.K3sExecArgs)
		os.Setenv("INSTALL_K3S_EXEC", opts.K3sExecArgs)
	}
	log.Info("Running k3s installation script")
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

	return cmd.Run()
}

func (i *installer) installManifest(manifest *types.PackageManifest) error {
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

	return nil
}
