package types

import "io"

// Node is an interface for preparing and managing a system that will run K3s.
type Node interface {
	MkdirAll(dir string) error
	GetFile(path string) (io.ReadCloser, error)
	WriteFile(rdr io.ReadCloser, destination string, mode string, size int64) error
	Execute(*ExecuteOptions) error
	Close() error
}

// ExecuteOptions represent options to an execute command on a node.
type ExecuteOptions struct {
	Env       map[string]string
	Command   string
	LogPrefix string
	Secrets   []string // strings to exclude from any debug logging
}
