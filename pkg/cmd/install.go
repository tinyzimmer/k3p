package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"

	v1 "github.com/tinyzimmer/k3p/pkg/build/package/v1"
	"github.com/tinyzimmer/k3p/pkg/cluster/node"
	"github.com/tinyzimmer/k3p/pkg/install"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

var (
	nodeRole           string
	installDocker      bool
	installOpts        *types.InstallOptions
	installConnectOpts *types.NodeConnectOptions
)

func init() {
	installOpts = &types.InstallOptions{}
	installConnectOpts = &types.NodeConnectOptions{}

	installCmd.Flags().BoolVarP(&installDocker, "docker", "D", false, "Install the package to a docker container on the local system.")
	installCmd.Flags().StringVarP(&installOpts.NodeName, "node-name", "n", "", "An optional name to give this node in the cluster")
	installCmd.Flags().BoolVar(&installOpts.AcceptEULA, "accept-eula", false, "Automatically accept any EULA included with the package")
	installCmd.Flags().StringVarP(&installOpts.ServerURL, "join", "j", "", "When installing an agent instance, the address of the server to join (e.g. https://myserver:6443)")
	installCmd.Flags().StringVarP(&nodeRole, "join-role", "r", "agent", `Specify whether to join the cluster as a "server" or "agent"`)
	installCmd.Flags().StringVarP(&installOpts.NodeToken, "join-token", "t", "", `When installing an additional agent or server instance, the node token to use.

For new agents, this can be retrieved with "k3p token get agent" or in 
"/var/lib/rancher/k3s/server/node-token" on any of the server instances.
For new servers, this value was either provided to or generated by 
"k3s install --init-ha" and can be retrieved from that server with 
"k3p token get server". When used with --init-ha, the provided token will 
be used for registering new servers, instead of one being generated.`)

	installCmd.Flags().StringVar(&installOpts.ResolvConf, "resolv-conf", "", `The path of a resolv-conf file to use when configuring DNS in the cluster.
When used with the --host flag, the path must reside on the remote system (this will change in the future).`)

	installCmd.Flags().StringVar(&installOpts.KubeconfigMode, "kubeconfig-mode", "", "The mode to set on the k3s kubeconfig. Default is to only allow root access")

	installCmd.Flags().StringVar(&installOpts.K3sExecArgs, "k3s-exec", "", `Extra arguments to pass to the k3s server or agent process, for more details see:
https://rancher.com/docs/k3s/latest/en/installation/install-options/server-config
`)
	installCmd.Flags().BoolVar(&installOpts.InitHA, "init-ha", false, `When set, this server will run with the --cluster-init flag to enable clustering, 
and a token will be generated for adding additional servers to the cluster with 
"--join-role server". You may optionally use the --join-token flag to provide a 
pre-generated one.`)

	// Remote installation options

	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	var defaultKeyArg string
	defaultKeyPath := path.Join(u.HomeDir, ".ssh", "id_rsa")
	if _, err := os.Stat(defaultKeyPath); err == nil {
		defaultKeyArg = defaultKeyPath
	}

	installCmd.Flags().StringVarP(&installConnectOpts.Address, "host", "H", "", "The IP or DNS name of a remote host to perform the installation against")
	installCmd.Flags().StringVarP(&installConnectOpts.SSHUser, "ssh-user", "u", u.Username, "The username to use when authenticating against the remote host")
	installCmd.Flags().StringVarP(&installConnectOpts.SSHKeyFile, "private-key", "k", defaultKeyArg, `The path to a private key to use when authenticating against the remote host, 
if not provided you will be prompted for a password`)
	installCmd.Flags().IntVarP(&installConnectOpts.SSHPort, "ssh-port", "p", 22, "The port to use when connecting to the remote host over SSH")

	rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install PACKAGE",
	Short: "Install the given package to the system",
	Long: `
The install command can be used to distribute a package built with "k3p build".

The command takes a single argument (with optional flags) of the filesystem path or web URL
where the package resides. Additional flags provide the ability to initialize clustering (HA),
join existing servers, or pass custom arguments to the k3s agent/server processes.

Example

	$> k3p install /path/on/filesystem.tar
	$> k3p install https://example.com/package.tar

When running on the local system like above, you will need to have root privileges. You can also 
direct the installation at a remote system over SSH via the --host flag. This will require the 
remote user having passwordless sudo available to them.

    $> k3p install package.tar --host 192.168.1.100 [SSH_FLAGS]

See the help below for additional information on available flags.
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// make sure we are root
		usr, err := user.Current()
		if err != nil {
			return err
		}

		// Retrieve the package from the command line
		pkg, err := getPackage(args[0])
		if err != nil {
			return err
		}

		// check the node role to make sure it's valid if relevant
		if installOpts.ServerURL != "" && nodeRole != "" {
			switch types.K3sRole(nodeRole) {
			case types.K3sRoleServer:
				installOpts.K3sRole = types.K3sRoleServer
			case types.K3sRoleAgent:
				installOpts.K3sRole = types.K3sRoleAgent
			default:
				return fmt.Errorf("%q is not a valid node role", nodeRole)
			}
		}

		var target types.Node
		if installConnectOpts.Address != "" {
			if installConnectOpts.SSHKeyFile == "" {
				fmt.Printf("Enter SSH Password for %s: ", installConnectOpts.SSHUser)
				bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return err
				}
				installConnectOpts.SSHPassword = string(bytePassword)
			}
			target, err = node.Connect(installConnectOpts)
			if err != nil {
				return err
			}
		} else {
			if usr.Uid != "0" {
				return errors.New("Local install must be run as root")
			}
			target = node.Local()
		}

		// run the installation
		err = install.New().Install(target, pkg, installOpts)
		if err != nil {
			return err
		}

		log.Info("The cluster has been installed. For additional details run `kubectl cluster-info`.")
		return nil
	},
}

func getPackage(path string) (types.Package, error) {
	if strings.HasPrefix(path, "http") {
		log.Info("Downloading the archive from", path)
		resp, err := http.Get(path)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		log.Info("Loading the archive")
		return v1.Load(resp.Body)
	}
	log.Info("Loading the archive")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return v1.Load(f)
}
