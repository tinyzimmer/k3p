package cache

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tinyzimmer/k3p/pkg/log"
)

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Download Cache Suite")
}

var mockCalled bool

func mockGet(url string) (*http.Response, error) {
	mockCalled = true
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(strings.NewReader("test")),
	}, nil
}

var _ = Describe("Download Cache", func() {
	// Send log output to the ginkgo writer
	log.LogWriter = GinkgoWriter

	// The DefaultCache should never be nil
	Expect(DefaultCache).ToNot(BeNil())
	// NoCache should default to false
	Expect(NoCache).To(BeFalse())

	Describe("Creating a new cache", func() {
		var (
			cache HTTPCache
		)
		JustBeforeEach(func() { cache = New("") })
		Context("When creating a new cache from a directory", func() {
			It("Should not be nil", func() {
				Expect(cache).ToNot(BeNil())
			})
		})
	})

	Describe("Get cache directory", func() {

		var (
			cache    *httpCache
			cacheDir string
		)

		JustBeforeEach(func() {
			cacheDir = cache.CacheDir()
		})

		Context("With a configured http cache", func() {
			BeforeEach(func() {
				cache = &httpCache{cacheDir: "test"}
			})
			It("Should return the configured cache directory", func() {
				Expect(cacheDir).To(Equal("test"))
			})
		})
	})

	Describe("Clean cache directory", func() {

		var (
			cache *httpCache
			err   error
		)

		JustBeforeEach(func() { err = cache.Clean() })

		Context("With no cache directory configured", func() {
			BeforeEach(func() {
				cache = &httpCache{}
			})
			It("Should return an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("With a valid cache directory configured", func() {
			BeforeEach(func() {
				tmpDir, err := ioutil.TempDir("", "")
				Expect(err).ToNot(HaveOccurred())
				cache = &httpCache{cacheDir: tmpDir}
			})
			It("Should not return an error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})

	})

	Describe("Downloading files using the cache", func() {
		var (
			tmpDir string
			cache  *httpCache
			url    string
			body   io.ReadCloser
			err    error
		)

		tmpDir, err = ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())
		cache = &httpCache{cacheDir: tmpDir, get: mockGet}

		JustBeforeEach(func() {
			mockCalled = false
			body, err = cache.Get(url)
		})

		Context("When retrieving a non-cached URL", func() {
			BeforeEach(func() { url = "https://example.com" })
			It("Should retrieve the body from the web and write it to the cache", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(body).ToNot(BeNil())
				Expect(mockCalled).To(BeTrue())
				files, err := ioutil.ReadDir(tmpDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(files)).To(Equal(1))
			})
		})

		Context("When retrieving a cached URL", func() {
			BeforeEach(func() {
				url = "https://example.com"
				_, err = cache.Get(url)
				Expect(err).ToNot(HaveOccurred())
			})
			It("Should retrieve the contents from the local cache", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(body).ToNot(BeNil())
				Expect(mockCalled).To(BeFalse())
			})
		})

		Context("When caching is disabled", func() {
			BeforeEach(func() {
				NoCache = true
				url = "https://example.com"
				_, err = cache.Get(url)
				Expect(err).ToNot(HaveOccurred())
			})
			It("Should always retrieve from the web", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(body).ToNot(BeNil())
				Expect(mockCalled).To(BeTrue())
			})
		})
	})
})
