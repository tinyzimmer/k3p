package util

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Utils Suite")
}

var _ = Describe("Utils", func() {

	// GetTempDir()
	Describe("Get Temp Directory", func() {

		var (
			cwd    string
			tmpDir string
			err    error
		)

		JustBeforeEach(func() {
			tmpDir, err = GetTempDir()
			Expect(err).ToNot(HaveOccurred())
			os.RemoveAll(tmpDir)
		})

		Context("When configured to the default", func() {
			It("Should return a directory under the system default", func() {
				Expect(path.Dir(tmpDir)).To(Equal(os.TempDir()))
			})
		})

		// This test assumes current directory is writable
		Context("When overwritten with a custom path", func() {
			BeforeEach(func() {
				cwd, err = os.Getwd()
				Expect(err).ToNot(HaveOccurred())
				TempDir = cwd
			})
			It("Should return a temp directory under the custom path", func() {
				Expect(path.Dir(tmpDir)).To(Equal(cwd))
			})
		})

	})

	// CalculateSHA256Sum
	Describe("Calculating SHA256 Sums", func() {
		var (
			shaSum string
			err    error
			body   io.ReadCloser
		)

		const (
			helloWorldSha = "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
		)

		JustBeforeEach(func() {
			shaSum, err = CalculateSHA256Sum(body)
		})

		Context("When passed the value 'hello world'", func() {
			BeforeEach(func() {
				body = ioutil.NopCloser(strings.NewReader("hello world"))
			})
			It("Should return the correct checksum", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(shaSum).To(Equal(helloWorldSha))
			})
		})

		Context("When passed a closed io.Reader", func() {
			BeforeEach(func() {
				body, _, err = os.Pipe()
				Expect(err).ToNot(HaveOccurred())
				body.Close()
			})
			It("Should return an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	// IsK8sObject
	Describe("Detecting K8s Objects from Unmarshaled Data", func() {
		var (
			data        map[string]interface{}
			isK8sObject bool
		)

		JustBeforeEach(func() { isK8sObject = IsK8sObject(data) })

		Context("When passed a valid kubernetes object", func() {
			BeforeEach(func() {
				data = map[string]interface{}{
					"kind":       "Pod",
					"apiVersion": "v1",
					"metadata": map[string]interface{}{
						"name": "test-pod",
					},
				}
			})
			Specify("That it is a valid object", func() {
				Expect(isK8sObject).To(BeTrue())
			})
		})

		Context("When passed an invalid kubernetes object", func() {
			BeforeEach(func() {
				data = map[string]interface{}{
					"hello":      "world",
					"apiVersion": "v1",
					"metadata": map[string]interface{}{
						"name": "invalid-pod",
					},
				}
			})
			Specify("That it is an invalid object", func() {
				Expect(isK8sObject).To(BeFalse())
			})
		})
	})

	// GenerateToken
	Describe("Generating Unique Tokens", func() {
		var (
			length int
			token  string
		)
		JustBeforeEach(func() { token = GenerateToken(length) })
		Context("When told to generate a 128 character token", func() {
			BeforeEach(func() { length = 128 })
			It("should return a token with 128 characters", func() {
				Expect(len(token)).To(Equal(128))
			})
		})
		Context("When told to generate a 256 character token", func() {
			BeforeEach(func() { length = 256 })
			It("should return a token with 256 characters", func() {
				Expect(len(token)).To(Equal(256))
			})
		})
	})

})
