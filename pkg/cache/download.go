package cache

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path"
	"strings"
	"time"

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
	cache := &httpCache{get: http.Get}
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

// HTTPCache is an interface for retrieving files from the internet while
// maintaining a local cache of received objects for use across CLI
// invocations. It is a very basic implementation that can be refactored
// in the future.
type HTTPCache interface {
	// CacheDir returns the current cache directory.
	CacheDir() string
	// Get retrieves the given URL from the cache, or the remote server if it
	// isn't already present locally.
	Get(url string) (io.ReadCloser, error)
	// GetIfOlder retrieves the given URL from the cache, or the remote server
	// if it isn't already present locally OR the provided duration since now
	// has expired.
	GetIfOlder(url string, dur time.Duration) (io.ReadCloser, error)
	// Clean will wipe the contents of the cache.
	Clean() error
}

// New creates a new HTTPCache using the given directory
func New(dir string) HTTPCache {
	return &httpCache{
		cacheDir: dir,
		get:      http.Get,
	}
}

type httpCache struct {
	cacheDir string
	get      func(url string) (*http.Response, error)
}

func (h *httpCache) CacheDir() string { return h.cacheDir }

func (h *httpCache) Clean() error {
	if h.cacheDir == "" {
		return errors.New("No cache directory detected")
	}
	log.Info("Wiping cache directory:", h.cacheDir)
	return os.RemoveAll(h.cacheDir)
}

func (h *httpCache) cachePathForURL(url string) (string, error) {
	cacheName, err := util.CalculateSHA256Sum(strings.NewReader(url))
	if err != nil {
		return "", err
	}
	return path.Join(h.cacheDir, cacheName), nil
}

func (h *httpCache) GetIfOlder(url string, dur time.Duration) (io.ReadCloser, error) {
	// If cache is setup check it first
	if !NoCache && h.cacheDir != "" {
		log.Debugf("Checking local cache for the contents of %q\n", url)
		cachePath, err := h.cachePathForURL(url)
		if err != nil {
			return nil, err
		}
		if fileExists(cachePath) {
			if dur > 0 {
				stat, err := os.Stat(cachePath)
				if err != nil {
					return nil, err
				}
				if stat.ModTime().Add(dur).After(time.Now()) {
					log.Debugf("Serving request from local cached item %q\n", cachePath)
					return os.Open(cachePath)
				}
				log.Debugf("Cached item for %q is older than %v\n", url, dur)
			} else {
				log.Debugf("Serving request from local cached item %q\n", cachePath)
				return os.Open(cachePath)
			}
		}
	}

	// We need to download the file
	log.Debug("Performing HTTP GET to", url)
	resp, err := h.get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("error retrieving %q: %s", url, string(body))
	}

	// If the cache is not configured, return the raw response body
	// so it can be closed properly by the caller.
	if NoCache || h.cacheDir == "" {
		log.Debug("Caching is not enabled, returning raw response object")
		return resp.Body, nil
	}

	// We have a local cache, save the object for future use
	defer resp.Body.Close()
	log.Debug("Calculating filename for new cache object")
	cachePath, err := h.cachePathForURL(url)
	if err != nil {
		return nil, err
	}
	log.Debugf("Writing %q to %q\n", url, cachePath)
	f, err := os.OpenFile(cachePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	return os.Open(cachePath)
}

func (h *httpCache) Get(url string) (io.ReadCloser, error) {
	return h.GetIfOlder(url, -1)
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
