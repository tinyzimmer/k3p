package node

import (
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/strslice"

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

func buildDockerEnv(opts *types.ExecuteOptions) strslice.StrSlice {
	out := make([]string, 0)
	for k, v := range opts.Env {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return strslice.StrSlice(out)
}

func buildDockerCmd(opts *types.ExecuteOptions) strslice.StrSlice {
	fields := strings.Fields(opts.Command)
	var cmd []string
	switch fields[len(fields)-1] {
	case string(types.K3sRoleAgent):
		cmd = []string{string(types.K3sRoleAgent)}
	default:
		cmd = []string{string(types.K3sRoleServer), "--tls-san", "0.0.0.0"}
	}
	for k, v := range opts.Env {
		if k == "INSTALL_K3S_EXEC" {
			cmd = append(cmd, strings.Fields(v)...)
		}
	}
	return strslice.StrSlice(cmd)
}
