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

	hm "github.com/evo-cloud/hmake/project"
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

func projectDir(project string, dir ...string) string {
	wd, err := os.Getwd()
	Expect(err).Should(Succeed())
	d := filepath.Join(wd, project)
	if dir != nil {
		d = filepath.Join(append([]string{d}, dir...)...)
	}
	return d
}

func hmakeCmd(project string, args ...string) *exec.Cmd {
	args = append([]string{"-C", projectDir(project)}, args...)
	return exec.Command(pathToHmake, args...)
}

func execHmake(project string, args ...string) *gexec.Session {
	session, err := gexec.Start(hmakeCmd(project, args...), GinkgoWriter, GinkgoWriter)
	Expect(err).Should(Succeed())
	return session
}

func waitHmake(project string, args ...string) *gexec.Session {
	session := execHmake(project, args...)
	session.Wait(15 * time.Minute)
	return session
}

func loadSummary(project string) hm.ExecSummary {
	proj, err := hm.LocateProjectFrom(projectDir(project), hm.RootFile)
	Expect(err).Should(Succeed())
	sum, err := proj.Summary()
	Expect(err).Should(Succeed())
	return sum
}

var _ = Describe("docker", func() {
	It("makes", func() {
		Eventually(waitHmake("docker", "-vR")).Should(gexec.Exit(0))
	})

	It("makes with correct env", func() {
		logfile := projectDir("docker-env", "test.log")
		os.Remove(logfile)
		Eventually(waitHmake("docker-env", "-vR")).Should(gexec.Exit(0))
		Expect(logfile).Should(BeAnExistingFile())
		data, err := ioutil.ReadFile(logfile)
		Expect(err).Should(Succeed())
		Expect(string(data)).To(Equal("TEST_VAL"))
	})

	It("makes with correct dir", func() {
		Eventually(waitHmake("docker-dir", "-vR")).Should(gexec.Exit(0))
	})

	It("aborts docker execution", func() {
		session := execHmake("docker-abort", "abort0", "-vR")
		time.Sleep(time.Second)
		session.Interrupt()
		session.Wait(30 * time.Second)
		Eventually(session).Should(gexec.Exit(1))
	})

	It("fix /etc/passwd", func() {
		Eventually(waitHmake("docker-user", "-vR")).Should(gexec.Exit(0))
	})

	It("rebuild target if changed", func() {
		Eventually(waitHmake("docker-cmds-change", "docker-cmd", "-vR")).Should(gexec.Exit(0))
		sum := loadSummary("docker-cmds-change").ByTarget("docker-cmd")
		Expect(sum).ShouldNot(BeNil())
		Expect(sum.Result).Should(Equal(hm.Success))

		Eventually(waitHmake("docker-cmds-change", "docker-cmd", "-v")).Should(gexec.Exit(0))
		sum = loadSummary("docker-cmds-change").ByTarget("docker-cmd")
		Expect(sum).ShouldNot(BeNil())
		Expect(sum.Result).Should(Equal(hm.Skipped))

		Eventually(waitHmake("docker-cmds-change", "-f", "HyperMake.changed", "docker-cmd", "-v")).Should(gexec.Exit(0))
		sum = loadSummary("docker-cmds-change").ByTarget("docker-cmd")
		Expect(sum).ShouldNot(BeNil())
		Expect(sum.Result).Should(Equal(hm.Success))
	})

	It("commit", func() {
		exec.Command("docker", "rmi", "hmake-test-commit:newtag", "hmake-test-commit:tag2").Run()
		// clean up
		defer func() {
			exec.Command("docker", "rmi", "hmake-test-commit:newtag", "hmake-test-commit:tag2").Run()
		}()
		Eventually(waitHmake("docker-commit", "test", "-vR")).Should(gexec.Exit(0))
	})

	It("docker-compose", func() {
		Eventually(waitHmake("docker-compose", "-vR")).Should(gexec.Exit(0))
	})

	Describe("exec", func() {
		It("exec", func() {
			Eventually(waitHmake("docker", "-x", "true")).Should(gexec.Exit(0))
		})

		It("not impact target result", func() {
			Eventually(waitHmake("docker", "exec", "-R")).Should(gexec.Exit(0))
			// try to fail --exec, can't use "false" because of docker bug
			//    docker create -it --name=test image /bin/false
			//    echo '' | docker start -a -i test
			//    echo $? => 0
			// However
			//    echo '' | /bin/false
			//    echo $? => 1
			// So use a non-exist command
			Eventually(waitHmake("docker", "-x", "non-exist")).Should(gexec.Exit(1))
			Eventually(waitHmake("docker", "exec")).Should(gexec.Exit(0))
			sum := loadSummary("docker").ByTarget("exec")
			Expect(sum).ShouldNot(BeNil())
			Expect(sum.Result).Should(Equal(hm.Skipped))
		})
	})

	Describe("wrapper-mode", func() {
		It("passthrough command line", func() {
			Eventually(waitHmake("wrapper-mode")).Should(gexec.Exit(1))
			Eventually(waitHmake("wrapper-mode", "t1")).Should(gexec.Exit(0))
		})
	})
})
