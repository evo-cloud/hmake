package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func TestEndToEnd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HyperMake E2E Suite")
}

var pathToHmake string

var _ = BeforeSuite(func() {
	var err error
	pathToHmake, err = gexec.Build("github.com/evo-cloud/hmake")
	Expect(err).Should(Succeed())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

var _ = Describe("docker", func() {
	It("makes", func() {
		wd, err := os.Getwd()
		Expect(err).Should(Succeed())
		cmd := exec.Command(pathToHmake, "-C", filepath.Join(wd, "docker"), "-v", "--debug")
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).Should(Succeed())
		session.Wait(15 * time.Minute)
		Eventually(session).Should(gexec.Exit(0))
	})
})
