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
	"strconv"
	"strings"
	"time"

	v1 "github.com/tinyzimmer/k3p/pkg/build/package/v1"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
	"github.com/tinyzimmer/k3p/pkg/util"
)

// New returns a new package installer.
func New() types.Installer { return &installer{getTarReader: getTarReader} }

type installer struct {
	getTarReader func(path string) (io.ReadCloser, int64, error)
}

func (i *installer) Install(target types.Node, opts *types.InstallOptions) error {
	log.Info("Copying the archive to the rancher installation directory")

	f, size, err := i.getTarReader(opts.TarPath)
	if err != nil {
		return err
	}

	if err := target.WriteFile(f, types.InstalledPackageFile, "0644", size); err != nil {
		return err
	}

	log.Info("Extracting the archive")
	// need a fresh tar reader, as the WriteFile will seek and close the first one
	f, err = target.GetFile(types.InstalledPackageFile)
	if err != nil {
		return err
	}

	pkg, err := v1.Load(f)
	if err != nil {
		return err
	}
	defer pkg.Close()

	// retrieve the full contents of the package
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

	// unpack the manifest into the appropriate locations
	if err := util.SyncPackageToNode(target, pkg); err != nil {
		return err
	}

	log.Info("Running k3s installation script")
	cmd, err := generateK3sInstallCommand(target, opts)
	if err != nil {
		return err
	}

	return target.Execute(cmd, "K3S")
}

func getTarReader(path string) (io.ReadCloser, int64, error) {
	if strings.HasPrefix(path, "http") {
		resp, err := http.Get(path)
		if err != nil {
			return nil, 0, err
		}
		size := resp.Header.Get("Content-Length")
		sizeInt, err := strconv.Atoi(size)
		if err != nil {
			return nil, 0, err
		}
		return resp.Body, int64(sizeInt), nil
	}
	stat, err := os.Stat(path)
	if err != nil {
		return nil, 0, err
	}
	f, err := os.Open(path)
	return f, stat.Size(), err
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

func generateK3sInstallCommand(target types.Node, opts *types.InstallOptions) (string, error) {
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
		// There needs to be a better place for this
		log.Debugf("Writing the contents of the server token to %s\n", types.ServerTokenFile)
		if err := target.WriteFile(ioutil.NopCloser(strings.NewReader(token)), types.ServerTokenFile, "0600", 128); err != nil {
			return "", err
		}
		cmd = cmd + fmt.Sprintf("K3S_TOKEN=%q ", token)
		// append --cluster-init to enable clustering (https://rancher.com/docs/k3s/latest/en/installation/ha-embedded/)
		opts.K3sExecArgs = opts.K3sExecArgs + " --cluster-init"
	} else if opts.ServerURL != "" && opts.NodeToken != "" {
		token = opts.NodeToken
		cmd = cmd + fmt.Sprintf("K3S_URL=%q K3S_TOKEN=%q ", opts.ServerURL, token)
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
	return cmd, nil
}
