package install

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
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

	if meta.Manifest.HasEULA() {
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

	if opts.InitHA {
		// append --cluster-init
		opts.K3sExecArgs = opts.K3sExecArgs + " --cluster-init"
		// Check if we need to generate an HA token
		if opts.NodeToken == "" {
			log.Info("Generating a node token for additional control-plane instances")
			token := util.GenerateToken(128)
			log.Debugf("Writing the contents of the server token to %s\n", types.ServerTokenFile)
			if err := target.WriteFile(ioutil.NopCloser(strings.NewReader(token)), types.ServerTokenFile, "0600", 128); err != nil {
				return err
			}
			opts.NodeToken = token
		}
	}

	execOpts := opts.ToExecOpts(pkg.GetMeta().GetPackageConfig())
	installedConfig := &types.InstallConfig{
		Variables: opts.Variables,
		Env:       execOpts.Env,
	}

	// unpack the manifest onto the node
	if err := util.SyncPackageToNode(target, pkg, installedConfig); err != nil {
		return err
	}

	// Install K3s
	log.Info("Running k3s installation script")
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
