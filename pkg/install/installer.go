package install

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
	"github.com/tinyzimmer/k3p/pkg/util"
)

// New returns a new package installer.
func New() types.Installer { return &installer{} }

type installer struct{}

func (i *installer) Install(target types.Node, pkg types.Package, opts *types.InstallOptions) error {
	defer pkg.Close()

	log.Info("Copying the archive to the rancher installation directory")

	archive, err := pkg.Archive()
	if err != nil {
		return err
	}
	if err := target.WriteFile(archive.Reader(), types.InstalledPackageFile, "0644", archive.Size()); err != nil {
		return err
	}

	// retrieve the meta to see if there is a EULA
	meta := pkg.GetMeta()

	if meta.Manifest.EULA != "" {
		// check package for a EULA
		eula := &types.Artifact{Name: types.ManifestEULAFile}
		if err := pkg.Get(eula); err == nil {
			// EULA found
			if err := promptEULA(eula, opts.AcceptEULA); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			// Error other than file not found
			return err
		}
	}

	// unpack the manifest onto the node
	if err := util.SyncPackageToNode(target, pkg); err != nil {
		return err
	}

	log.Info("Running k3s installation script")
	execOpts, err := generateK3sInstallOpts(target, opts)
	if err != nil {
		return err
	}

	return target.Execute(execOpts)
}

func promptEULA(eula *types.Artifact, autoAccept bool) error {
	if autoAccept {
		return nil
	}
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
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Do you accept the terms of the EULA? [y/N] ")
		scanner.Scan()
		text := scanner.Text()
		switch strings.ToLower(text) {
		case "y":
			time.Sleep(time.Second)
			return nil
		case "n":
			return errors.New("EULA was declined")
		default:
			fmt.Printf("%q is not a valid response, choose 'y' or 'n' \n", text)
		}
	}
}

func generateK3sInstallOpts(target types.Node, opts *types.InstallOptions) (*types.ExecuteOptions, error) {
	var token string

	env := map[string]string{
		"INSTALL_K3S_SKIP_DOWNLOAD": "true",
	}

	if opts.NodeName != "" {
		env["K3S_NODE_NAME"] = opts.NodeName
	}

	if opts.ResolvConf != "" {
		env["K3S_RESOLV_CONF"] = opts.ResolvConf
	}

	if opts.KubeconfigMode != "" {
		env["K3S_KUBECONFIG_MODE"] = opts.KubeconfigMode
	}

	// these are mutually exclusive, should be better documented
	if opts.InitHA {
		if opts.NodeToken == "" {
			log.Info("Generating a node token for additional control-plane instances")
			token = util.GenerateToken(128)
		}
		// There needs to be a better place for this
		log.Debugf("Writing the contents of the server token to %s\n", types.ServerTokenFile)
		if err := target.WriteFile(ioutil.NopCloser(strings.NewReader(token)), types.ServerTokenFile, "0600", 128); err != nil {
			return nil, err
		}
		env["K3S_TOKEN"] = token
		// append --cluster-init to enable clustering (https://rancher.com/docs/k3s/latest/en/installation/ha-embedded/)
		opts.K3sExecArgs = opts.K3sExecArgs + " --cluster-init"
	} else if opts.ServerURL != "" && opts.NodeToken != "" {
		token = opts.NodeToken
		env["K3S_URL"] = opts.ServerURL
		env["K3S_TOKEN"] = token
	}

	if opts.K3sExecArgs != "" {
		env["INSTALL_K3S_EXEC"] = opts.K3sExecArgs
	}

	secrets := []string{}
	if token != "" {
		secrets = []string{token}
	}

	return &types.ExecuteOptions{
		Env:       env,
		Command:   fmt.Sprintf("sh %q %s", path.Join(types.K3sScriptsDir, "install.sh"), string(opts.K3sRole)),
		LogPrefix: "K3S",
		Secrets:   secrets,
	}, nil
}
