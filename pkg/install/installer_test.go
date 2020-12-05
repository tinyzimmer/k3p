package install

import (
	"io"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "github.com/tinyzimmer/k3p/pkg/build/archive/v1"
	"github.com/tinyzimmer/k3p/pkg/cluster/node"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

func getMockReader(string) (io.ReadCloser, int64, error) { return v1.Mock(), v1.MockSize(), nil }
func mockInstaller() *installer                          { return &installer{getMockReader} }

func TestUtils(t *testing.T) {
	log.LogWriter = GinkgoWriter
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installer Suite")
}

var _ = Describe("Installer", func() {
	var (
		err  error
		opts types.InstallOptions
	)

	target := node.Mock()
	defer target.Close()

	JustBeforeEach(func() {
		err = mockInstaller().Install(target, &opts)
	})

	Context("With no error conditions present", func() {
		It("Should succeed", func() {
			Expect(err).ToNot(HaveOccurred())
		})
	})

	// TODO: More tests
})
