package e2e

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

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

	It("makes with correct env", func() {
		wd, err := os.Getwd()
		Expect(err).Should(Succeed())
		logfile := filepath.Join(wd, "docker-env", "test.log")
		os.Remove(logfile)
		cmd := exec.Command(pathToHmake, "-C", filepath.Join(wd, "docker-env"), "-v", "--debug")
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).Should(Succeed())
		session.Wait(15 * time.Minute)
		Eventually(session).Should(gexec.Exit(0))
		Expect(logfile).Should(BeAnExistingFile())
		data, err := ioutil.ReadFile(logfile)
		Expect(err).Should(Succeed())
		Expect(string(data)).To(Equal("TEST_VAL"))
	})

	It("makes with correct dir", func() {
		wd, err := os.Getwd()
		Expect(err).Should(Succeed())
		cmd := exec.Command(pathToHmake, "-C", filepath.Join(wd, "docker-dir"), "-v", "--debug")
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).Should(Succeed())
		session.Wait(15 * time.Minute)
		Eventually(session).Should(gexec.Exit(0))
	})

	It("aborts docker execution", func() {
		wd, err := os.Getwd()
		Expect(err).Should(Succeed())
		cmd := exec.Command(pathToHmake, "-C", filepath.Join(wd, "docker-abort"), "abort0", "-v", "--debug")
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).Should(Succeed())
		time.Sleep(time.Second)
		session.Interrupt()
		session.Wait(5 * time.Second)
		Eventually(session).Should(gexec.Exit(1))
	})
})
