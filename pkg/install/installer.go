package install

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	v1 "github.com/tinyzimmer/k3p/pkg/build/archive/v1"
	"github.com/tinyzimmer/k3p/pkg/cluster/node"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
	"github.com/tinyzimmer/k3p/pkg/util"
)

// New returns a new package installer.
func New() types.Installer { return &installer{} }

type installer struct{}

func (i *installer) Install(opts *types.InstallOptions) error {
	system := node.Local()

	log.Info("Copying the package to the rancher installation directory")

	f, err := getTarReader(opts.TarPath)
	if err != nil {
		return err
	}

	if err := system.WriteFile(f, types.InstalledPackageFile, "0644", 0); err != nil {
		return err
	}

	log.Info("Extracting the archive")
	pkg, err := v1.Load(types.InstalledPackageFile)
	if err != nil {
		return err
	}
	defer pkg.Close()

	// check package for a EULA
	eula := &types.Artifact{Name: "EULA.txt"}
	if err := pkg.Get(eula); err == nil {
		// EULA found
		if err := promptEULA(eula, opts.AcceptEULA); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		// Error other than file not found
		return err
	}

	// retrieve the full contents of the package
	manifest, err := pkg.GetManifest()
	if err != nil {
		return err
	}

	// unpack the manifest into the appropriate locations
	if err := node.SyncManifestToNode(system, manifest); err != nil {
		return err
	}

	log.Info("Running k3s installation script")
	cmd := generateK3sInstallCommand(opts)

	return system.Execute(cmd, "K3S")
}

func getTarReader(path string) (io.ReadCloser, error) {
	if strings.HasPrefix(path, "http") {
		resp, err := http.Get(path)
		if err != nil {
			return nil, err
		}
		return resp.Body, nil
	}
	return os.Open(path)
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

func generateK3sInstallCommand(opts *types.InstallOptions) string {
	var token string

	cmd := `INSTALL_K3S_SKIP_DOWNLOAD="true" `

	if opts.NodeName != "" {
		cmd = cmd + fmt.Sprintf("K3S_NODE_NAME=%q ", opts.NodeName)
	}

	if opts.ResolvConf != "" {
		cmd = cmd + fmt.Sprintf("K3S_RESOLV_CONF=%q ", opts.ResolvConf)
	}

	if opts.KubeconfigMode != "" {
		cmd = cmd + fmt.Sprintf("K3S_KUBECONFIG_MODE=%q ", opts.KubeconfigMode)
	}

	// these are mutually exclusive, should be better documented
	if opts.InitHA {
		if opts.NodeToken == "" {
			log.Info("Generating a node token for additional control-plane instances")
			token = util.GenerateToken(128)
		}
		log.Debugf("Writing the contents of the token to %s\n", types.ServerTokenFile)
		if err := ioutil.WriteFile(types.ServerTokenFile, []byte(strings.TrimSpace(token)), 0600); err != nil {
			// TODO: error handling, this is technically important
			log.Error("Failed to write the server join token to the filesystem. Be sure to copy it down for future reference.")
			log.Error(err)
		}
		log.Info("You can join new servers to the control-plane with the following token:", token) // with above fixed, don't log, just make available in the future
		cmd = cmd + fmt.Sprintf("K3S_TOKEN=%q ", token)
		// append --cluster-init to enable clustering (https://rancher.com/docs/k3s/latest/en/installation/ha-embedded/)
		opts.K3sExecArgs = opts.K3sExecArgs + " --cluster-init"
	} else if opts.ServerURL != "" && opts.NodeToken != "" {
		cmd = cmd + fmt.Sprintf("K3S_URL=%q K3S_TOKEN=%q ", opts.ServerURL, opts.NodeToken)
	}

	if opts.K3sExecArgs != "" {
		cmd = cmd + fmt.Sprintf("INSTALL_K3S_EXEC=%q ", opts.K3sExecArgs)
	}

	cmd = cmd + fmt.Sprintf("sudo -E sh %s %s", path.Join(types.K3sScriptsDir, "install.sh"), string(opts.K3sRole))
	loggedCmd := cmd
	if token != "" {
		loggedCmd = strings.Replace(loggedCmd, token, "<redacted>", -1)
	}
	log.Debug("Generated K3s installation command:", loggedCmd)
	return cmd
}
