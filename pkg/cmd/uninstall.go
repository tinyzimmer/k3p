package cmd

import (
	"github.com/spf13/cobra"

	"github.com/tinyzimmer/k3p/pkg/cluster/node"
	"github.com/tinyzimmer/k3p/pkg/log"
)

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

func completeClusters(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	log.Verbose = false
	clusters, err := node.ListDockerClusters()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return clusters, cobra.ShellCompDirectiveDefault
}

var uninstallCmd = &cobra.Command{
	Use:               "uninstall",
	Short:             "Uninstall a k3p package (currently only for docker)",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeClusters,
	RunE: func(cmd *cobra.Command, args []string) error {
		uninstallName := args[0]
		nodes, err := node.LoadDockerCluster(uninstallName)
		if err != nil {
			return err
		}
		if len(nodes) == 0 {
			log.Info("No running clusters found for", uninstallName)
			return nil
		}
		log.Info("Removing docker cluster", uninstallName)
		for _, dockerNode := range nodes {
			defer dockerNode.Close()
			if err := dockerNode.RemoveAll(); err != nil {
				return err
			}
		}
		log.Info("Removing docker network", uninstallName)
		return node.DeleteDockerNetwork(uninstallName)
	},
}
