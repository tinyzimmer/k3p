package install

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/tinyzimmer/k3p/pkg/images/registry"
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

	cfg := pkg.GetMeta().DeepCopy().GetPackageConfig()
	log.Debugf("Package configuration: %+v\n", cfg)
	if cfg != nil {
		if err := cfg.ApplyVariables(opts.Variables); err != nil {
			return err
		}
	}
	execOpts := opts.ToExecOpts(cfg)

	if opts.InitHA {
		// append --cluster-init
		execOpts.Env["INSTALL_K3S_EXEC"] = execOpts.Env["INSTALL_K3S_EXEC"] + " --cluster-init"
		// Check if we need to generate an HA token
		if opts.NodeToken == "" {
			log.Info("Generating a node token for additional control-plane instances")
			token := util.GenerateToken(128)
			log.Debugf("Writing the contents of the server token to %s\n", types.ServerTokenFile)
			if err := target.WriteFile(ioutil.NopCloser(strings.NewReader(token)), types.ServerTokenFile, "0600", 128); err != nil {
				return err
			}
			execOpts.Env["K3S_TOKEN"] = token
			execOpts.Secrets = append(execOpts.Secrets, token)
		}
	}

	if meta.ImageBundleFormat == types.ImageBundleRegistry {
		log.Info("Package was generated with private registry")
		if err := setupPrivateRegistry(target, meta, opts); err != nil {
			return err
		}
	}

	installedConfig := &types.InstallConfig{InstallOptions: opts}
	log.Debugf("Built installation config %+v\n", installedConfig)

	// unpack the manifest onto the node
	if err := util.SyncPackageToNode(target, pkg, installedConfig); err != nil {
		return err
	}

	// Install K3s
	if target.GetType() != types.NodeDocker {
		// let's not lie to the user when we are doing docker installs
		log.Info("Running k3s installation script")
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

func setupPrivateRegistry(target types.Node, meta *types.PackageMeta, opts *types.InstallOptions) error {
	registryManifestPath := path.Join(types.K3sManifestsDir, "private-registry")

	log.Info("Setting up registry TLS")
	caCert, secrets, err := registry.GenerateRegistryTLSSecrets(&types.RegistryTLSOptions{
		Name:                meta.GetName(),
		RegistryTLSCertFile: opts.RegistryTLSCertFile,
		RegistryTLSKeyFile:  opts.RegistryTLSKeyFile,
		RegistryTLSCAFile:   opts.RegistryTLSCAFile,
	})
	if err != nil {
		return err
	}
	if err := target.WriteFile(nopCloser(caCert), registry.RegistryCAPath, "0644", size(caCert)); err != nil {
		return err
	}
	if err := target.WriteFile(nopCloser(secrets), path.Join(registryManifestPath, "registry-tls-secrets.yaml"), "0644", size(secrets)); err != nil {
		return err
	}

	log.Info("Writing secrets for registry authentication")
	if opts.RegistrySecret == "" {
		log.Info("Generating password for registry authentication")
		opts.RegistrySecret = util.GenerateToken(16)
	}
	authSecret, err := registry.GenerateRegistryAuthSecret(opts.RegistrySecret)
	if err != nil {
		return err
	}
	if err := target.WriteFile(nopCloser(authSecret), path.Join(registryManifestPath, "registry-auth-secret.yaml"), "0644", size(authSecret)); err != nil {
		return err
	}

	log.Info("Writing deployments and services for the private registry")
	svcs, err := registry.GenerateRegistryServices(opts.GetRegistryNodePort())
	if err != nil {
		return err
	}
	deployments, err := registry.GenerateRegistryDeployments(meta.GetRegistryImageName())
	if err != nil {
		return err
	}
	if err := target.WriteFile(nopCloser(svcs), path.Join(registryManifestPath, "registry-services.yaml"), "0644", size(svcs)); err != nil {
		return err
	}
	if err := target.WriteFile(nopCloser(deployments), path.Join(registryManifestPath, "registry-deployments.yaml"), "0644", size(deployments)); err != nil {
		return err
	}

	log.Info("Writing containerd configuration for the private registry")
	registryConf, err := registry.GenerateRegistriesYaml(opts.RegistrySecret, opts.GetRegistryNodePort())
	if err != nil {
		return err
	}
	if err := target.WriteFile(nopCloser(registryConf), types.K3sRegistriesYamlPath, "0644", size(registryConf)); err != nil {
		return err
	}

	return nil
}

func size(b []byte) int64 { return int64(len(b)) }

func nopCloser(b []byte) io.ReadCloser { return ioutil.NopCloser(bytes.NewReader(b)) }
