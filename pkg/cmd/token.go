package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tinyzimmer/k3p/pkg/types"
	"github.com/tinyzimmer/k3p/pkg/util"
)

var generatedTokenLength int

func init() {
	tokenGenerateCmd.Flags().IntVarP(&generatedTokenLength, "length", "l", 128, "The length of the token to generate")

	tokenCmd.AddCommand(tokenGetCmd)
	tokenCmd.AddCommand(tokenGenerateCmd)
	rootCmd.AddCommand(tokenCmd)
}

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Token retrieval and generation commands",
}

var tokenGetCmd = &cobra.Command{
	Use:   "get TOKEN_TYPE",
	Short: "Retrieve a k3s token",
	Long: `
Retrieves the token for joining either a new "agent" or "server" to the cluster.

The "agent" token can be retrieved from any of the server instances, while the "server" token
can only be retrieved on the server where "k3p install" was run with "--init-ha".
`,
	Args:      cobra.ExactValidArgs(1),
	ValidArgs: []string{"agent", "server"},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "agent":
			token, err := ioutil.ReadFile(types.AgentTokenFile)
			if err != nil {
				if os.IsNotExist(err) {
					return errors.New("The K3s server does not appear to be installed to the system")
				}
				return err
			}
			fmt.Println(strings.TrimSpace(string(token)))
		case "server":
			token, err := ioutil.ReadFile(types.ServerTokenFile)
			if err != nil {
				if os.IsNotExist(err) {
					return errors.New("This system does not appear to have been initialized with --init-ha")
				}
				return err
			}
			fmt.Println(strings.TrimSpace(string(token)))
		}
		return nil
	},
}

var tokenGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generates a token that can be used for initializing HA installations",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(util.GenerateToken(generatedTokenLength))
	},
}
