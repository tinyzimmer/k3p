package node

import (
	"fmt"
	"strings"

	"github.com/tinyzimmer/k3p/pkg/types"
)

func redactSecrets(logLine string, secrets []string) string {
	for _, secret := range secrets {
		logLine = strings.Replace(logLine, secret, "<redacted>", -1)
	}
	return logLine
}

func buildCmdFromExecOpts(opts *types.ExecuteOptions) string {
	var cmd string
	for k, v := range opts.Env {
		cmd = cmd + fmt.Sprintf("%s=%q ", k, v)
	}
	cmd = cmd + "sudo -E " + opts.Command
	return cmd
}
