package node

import (
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

// Local returns a new Node pointing at the local system.
func Local() types.Node {
	return &localNode{}
}

type localNode struct{}

func (l *localNode) MkdirAll(dir string) error {
	log.Debugf("Ensuring local system directory %q with mode 0755\n", dir)
	return os.MkdirAll(dir, 0755)
}

func (l *localNode) Close() error { return nil }

func (l *localNode) GetFile(f string) (io.ReadCloser, error) { return os.Open(f) }

// size is ignored for local nodes
func (l *localNode) WriteFile(rdr io.ReadCloser, dest string, mode string, size int64) error {
	log.Debugf("Writing file to local system at %q with mode %q\n", dest, mode)
	defer rdr.Close()
	if err := l.MkdirAll(path.Dir(dest)); err != nil {
		return err
	}
	u, err := strconv.ParseUint("0755", 0, 16)
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

func (l *localNode) Execute(cmd string, logPrefix string) error {
	c := exec.Command("/bin/sh", "-c", cmd)
	outPipe, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	errPipe, err := c.StderrPipe()
	if err != nil {
		return err
	}
	go log.TailReader(logPrefix, outPipe)
	go log.TailReader(logPrefix, errPipe)
	return c.Run()
}
