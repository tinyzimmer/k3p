package cache

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/util"
)

// NoCache can be set by a CLI flag to signal that fresh copies should be downloaded
// for every request.
var NoCache bool

// DefaultCache is the default http cache configured at init. It should be used for most
// operations.
var DefaultCache HTTPCache

func init() {
	cache := &httpCache{}
	defer func() { DefaultCache = cache }()
	usr, err := user.Current()
	if err != nil {
		log.Error(err)
		return
	}
	cacheDir := path.Join(usr.HomeDir, ".k3p", "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Error(err)
		return
	}
	cache.cacheDir = cacheDir
}

// HTTPCache is an object for retrieving files from the internet while
// maintaining a local cache of received objects for use across CLI
// invocations. It is a very basic implementation that can be refactored
// in the future.
type HTTPCache interface {
	// CacheDir returns the current cache directory.
	CacheDir() string
	// Get retrieves the given URL from the cache, or the remote server if it
	// isn't already present locally.
	Get(url string) (io.ReadCloser, error)
	// Clean will wipe the contents of the cache.
	Clean() error
}

// New creates a new HTTPCache using the given directory
func New(dir string) HTTPCache {
	return &httpCache{cacheDir: dir}
}

type httpCache struct {
	cacheDir string
}

func (h *httpCache) CacheDir() string { return h.cacheDir }

func (h *httpCache) Clean() error {
	if h.cacheDir == "" {
		return errors.New("No cache directory detected")
	}
	log.Info("Wiping cache directory:", h.cacheDir)
	return os.RemoveAll(h.cacheDir)
}

func (h *httpCache) Get(url string) (io.ReadCloser, error) {
	// If cache is setup check it first
	if !NoCache && h.cacheDir != "" {
		log.Debugf("Checking local cache for the contents of %q", url)
		cacheName, err := util.CalculateSHA256Sum(strings.NewReader(url))
		if err != nil {
			return nil, err
		}
		cachePath := path.Join(h.cacheDir, cacheName)
		if fileExists(cachePath) {
			log.Debug("Serving request from local cache")
			return os.Open(cachePath)
		}
	}

	// We need to download the file
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	// If the cache is not configured, return the raw response body
	// so it can be closed properly by the caller.
	if NoCache || h.cacheDir == "" {
		log.Debug("Caching is not enabled, returning raw response object")
		return resp.Body, nil
	}

	// setup a tee reader to cache the contents and populate a new buffer for return
	var buf bytes.Buffer
	out := ioutil.NopCloser(&buf)
	tee := io.TeeReader(resp.Body, &buf)
	defer resp.Body.Close()

	// We have a local cache, save the object for future use
	cacheName, err := util.CalculateSHA256Sum(strings.NewReader(url))
	if err != nil {
		return nil, err
	}
	cachePath := path.Join(h.cacheDir, cacheName)
	log.Debugf("Writing %q to %q", url, cachePath)
	f, err := os.OpenFile(cachePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(f, tee); err != nil {
		return nil, err
	}

	return out, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		log.Error("Unexpected error from os.Stat:", err)
		return false
	}
	return !info.IsDir()
}
