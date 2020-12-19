package node

import (
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
	"github.com/tinyzimmer/k3p/pkg/util"
)

// Local returns a new Node pointing at the local system.
func Local() types.Node {
	return &localNode{}
}

type localNode struct{}

func (l *localNode) GetType() types.NodeType { return types.NodeLocal }

func (l *localNode) MkdirAll(dir string) error {
	log.Debugf("Ensuring local system directory %q with mode 0755\n", dir)
	return os.MkdirAll(dir, 0755)
}

func (l *localNode) Close() error { return nil }

func (l *localNode) GetFile(f string) (io.ReadCloser, error) { return os.Open(f) }

// size is ignored for local nodes
func (l *localNode) WriteFile(rdr io.ReadCloser, dest string, mode string, size int64) error {
	defer rdr.Close()
	if err := l.MkdirAll(path.Dir(dest)); err != nil {
		return err
	}
	log.Debugf("Writing file to local system at %q with mode %q\n", dest, mode)
	u, err := strconv.ParseUint(mode, 0, 16)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE, os.FileMode(u))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, rdr)
	return err
}

func (l *localNode) Execute(opts *types.ExecuteOptions) error {
	cmd := buildCmdFromExecOpts(opts)
	log.Debug("Executing command on local system:", redactSecrets(cmd, opts.Secrets))
	c := exec.Command("/bin/sh", "-c", cmd)
	outPipe, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	errPipe, err := c.StderrPipe()
	if err != nil {
		return err
	}
	go log.LevelReader(log.LevelInfo, outPipe)
	go log.LevelReader(log.LevelError, errPipe)
	return c.Run()
}

func (l *localNode) GetK3sAddress() (string, error) {
	addr, err := util.GetExternalAddressForProcess("k3s-server")
	if err != nil {
		return "", err
	}
	return addr.String(), nil
}
