package cmd

import (
	"github.com/spf13/cobra"

	"github.com/tinyzimmer/k3p/pkg/cluster/node"
	"github.com/tinyzimmer/k3p/pkg/log"
)

var (
	uninstallName string
)

func init() {
	uninstallCmd.Flags().StringVarP(&uninstallName, "name", "n", "", "The name of the package to uninstall (required for docker)")
	if err := uninstallCmd.MarkFlagRequired("name"); err != nil {
		log.Error(err)
	}

	uninstallCmd.RegisterFlagCompletionFunc("name", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		clusters, err := node.ListDockerClusters()
		if err != nil {
			return nil, cobra.ShellCompDirectiveDefault
		}
		return clusters, cobra.ShellCompDirectiveDefault
	})

	rootCmd.AddCommand(uninstallCmd)
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall a k3p package (currently only for docker)",
	RunE: func(cmd *cobra.Command, args []string) error {
		nodes, err := node.LoadDockerCluster(uninstallName)
		if err != nil {
			return err
		}
		if len(nodes) == 0 {
			log.Info("No running clusters found for", uninstallName)
			return nil
		}
		for _, node := range nodes {
			defer node.Close()
			if addr, err := node.GetK3sAddress(); err == nil { // it's always nil for docker
				log.Info("Removing container and volumes for", addr)
			}
			if err := node.RemoveAll(); err != nil {
				return err
			}
		}
		log.Info("Removing docker network", uninstallName)
		return node.DeleteDockerNetwork(uninstallName)
	},
}
