package types

import (
	"io"
	"strconv"
	"strings"
)

// NodeType represents a type of node
type NodeType string

const (
	// NodeLocal represents the local system
	NodeLocal NodeType = "local"
	// NodeRemote represents a remote node over SSH
	NodeRemote NodeType = "remote"
	// NodeDocker represents a docker container node
	NodeDocker NodeType = "docker"
)

// Node is an interface for preparing and managing a system that will run K3s.
type Node interface {
	// GetType should be implemented by every node and return one of the types above
	GetType() NodeType
	// MkdirAll should ensure the given directory on the node
	MkdirAll(dir string) error
	// GetFile should retrieve the given file on the node
	GetFile(path string) (io.ReadCloser, error)
	// WriteFile should write the contents of the given reader to destination on the node,
	// and set its mode and size accordingly.
	WriteFile(rdr io.ReadCloser, destination string, mode string, size int64) error
	// Execute should execute a command on the node. This function should probably be renamed/repurposed
	// to StartK3s or something as that is all it is used for, and will make more sense in the
	// context of docker.
	Execute(*ExecuteOptions) error
	// GetK3sAddress should return the address where the k3s server is listening for connections
	// on this node.
	GetK3sAddress() (string, error)
	// Close should close any open connections to the node and perform any necessary cleanup.
	Close() error
}

// ExecuteOptions represent options to an execute command on a node.
type ExecuteOptions struct {
	// Environment variables to set for the process
	Env map[string]string
	// The command to run
	Command string
	// Secret strings to filter from any logging output
	Secrets []string
}

// GetAPIPort returns the API port configured for these ExecuteOptions. This is a bit of a hack
// for the fact that the user is allowed to pass in raw k3s exec strings. A lot of this needs to
// be refactored.
// What made this necessary is now taken care of, it may be safe to reevaluate this now.
func (e *ExecuteOptions) GetAPIPort() int {
	for k, v := range e.Env {
		if k == "INSTALL_K3S_EXEC" {
			fields := strings.Fields(v)
			for idx, arg := range fields {
				if strings.HasPrefix(arg, "--https-listen-port=") {
					if port, err := strconv.Atoi(strings.TrimPrefix(arg, "--https-listen-port=")); err == nil {
						return port
					}
				}
				if arg == "--https-listen-port" {
					if port, err := strconv.Atoi(fields[idx+1]); err == nil {
						return port
					}
				}
			}
		}
	}
	return 6443
}
