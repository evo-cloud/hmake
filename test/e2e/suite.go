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
		cmd := exec.Command(pathToHmake, "-C", filepath.Join(wd, "docker"), "-vR")
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
		cmd := exec.Command(pathToHmake, "-C", filepath.Join(wd, "docker-env"), "-vR")
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
		cmd := exec.Command(pathToHmake, "-C", filepath.Join(wd, "docker-dir"), "-vR")
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).Should(Succeed())
		session.Wait(15 * time.Minute)
		Eventually(session).Should(gexec.Exit(0))
	})

	It("aborts docker execution", func() {
		wd, err := os.Getwd()
		Expect(err).Should(Succeed())
		cmd := exec.Command(pathToHmake, "-C", filepath.Join(wd, "docker-abort"), "abort0", "-vR")
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).Should(Succeed())
		time.Sleep(time.Second)
		session.Interrupt()
		session.Wait(30 * time.Second)
		Eventually(session).Should(gexec.Exit(1))
	})

	It("fix /etc/passwd", func() {
		wd, err := os.Getwd()
		Expect(err).Should(Succeed())
		cmd := exec.Command(pathToHmake, "-C", filepath.Join(wd, "docker-user"), "-vR")
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).Should(Succeed())
		session.Wait(15 * time.Minute)
		Eventually(session).Should(gexec.Exit(0))
	})

	It("commit", func() {
		wd, err := os.Getwd()
		Expect(err).Should(Succeed())
		exec.Command("docker", "rmi", "hmake-test-commit:newtag", "hmake-test-commit:tag2").Run()
		// clean up
		defer func() {
			exec.Command("docker", "rmi", "hmake-test-commit:newtag", "hmake-test-commit:tag2").Run()
		}()
		cmd := exec.Command(pathToHmake, "-C", filepath.Join(wd, "docker-commit"), "test", "-vR")
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).Should(Succeed())
		session.Wait(30 * time.Second)
		Eventually(session).Should(gexec.Exit(0))
	})

})
