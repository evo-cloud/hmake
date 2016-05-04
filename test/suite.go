package test

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	hm "github.com/evo-cloud/hmake/project"
)

// Test environment related
var (
	ProjectDir  string
	TestDir     string
	FixturesDir string
)

// Fixtures returns the directory under fixtures
func Fixtures(dirs ...string) string {
	return filepath.Join(FixturesDir, filepath.Join(dirs...))
}

// Samples returns the directory with samples
func Samples(dirs ...string) string {
	return Fixtures("samples", filepath.Join(dirs...))
}

// LoadProject loads specified project and assert success
func LoadProject(base, file string) *hm.Project {
	proj := &hm.Project{BaseDir: base}
	Expect(proj.Load(file)).Should(Not(BeNil()))
	Expect(proj.Resolve()).Should(Succeed())
	Expect(proj.Finalize()).Should(Succeed())
	return proj
}

// Runner is the test exec-driver
func Runner(task *hm.Task) (hm.TaskResult, error) {
	return hm.Success, nil
}

type testSetting struct {
	TopLevel  string `json:"toplevel"`
	TopLevel1 string `json:"toplevel1"`
	Dict      struct {
		Key  string `json:"key"`
		Key1 string `json:"key1"`
	} `json:"dict"`
}

var _ = BeforeSuite(func() {
	ProjectDir = os.Getenv("HMAKE_PROJECT_DIR")
	if ProjectDir == "" {
		proj, err := hm.LocateProject()
		Expect(err).NotTo(HaveOccurred())
		Expect(proj.Name).To(Equal("hmake"))
		ProjectDir = proj.BaseDir
	}
	Expect(ProjectDir).To(BeADirectory())
	TestDir = filepath.Join(ProjectDir, "test")
	FixturesDir = filepath.Join(TestDir, "fixtures")
	Expect(FixturesDir).To(BeADirectory())

	hm.DefaultExecDriver = "test"
	hm.RegisterExecDriver("test", Runner)
})

var _ = Describe("HyperMake", func() {
	Describe("Project", func() {
		It("fails to load file", func() {
			_, err := hm.LoadFile(Samples(), "invalid-yaml.hmake")
			Expect(err).ShouldNot(Succeed())
			_, err = hm.LoadFile(Samples(), "non-exist.hmake")
			Expect(err).ShouldNot(Succeed())
		})

		It("checks the format", func() {
			_, err := hm.LoadFile(Samples(), "missing-format.hmake")
			Expect(err).
				Should(MatchError(ContainSubstring("unsupported format")))
			_, err = hm.LoadFile(Samples(), "bad-format.hmake")
			Expect(err).
				Should(MatchError(ContainSubstring("unsupported format")))
		})

		It("locates the project", func() {
			proj, err := hm.LoadProjectFrom(Fixtures(
				"project0", "subproject", "subdir", "subdir2"))
			Expect(err).Should(Succeed())
			Expect(proj.Name).To(Equal("subdir"))
			proj, err = hm.LoadProjectFrom(Fixtures(
				"project0", "subproject", "subdir"))
			Expect(err).Should(Succeed())
			Expect(proj.Name).To(Equal("subdir"))
			proj, err = hm.LoadProjectFrom(Fixtures(
				"project0", "subproject"))
			Expect(err).Should(Succeed())
			Expect(proj.Name).To(Equal("project0"))
			_, err = hm.LoadProjectFrom("/")
			Expect(err).Should(MatchError(os.ErrNotExist))
		})

		It("detects the cyclic deps", func() {
			proj := &hm.Project{BaseDir: Samples()}
			Expect(proj.Load("cyclic-deps.hmake")).ShouldNot(BeNil())
			Expect(proj.Name).To(Equal("cyclic-deps"))
			Expect(proj.Resolve()).Should(Succeed())
			Expect(proj.Finalize()).
				Should(MatchError(ContainSubstring("cyclic dependency")))
		})

		It("dependency not defined", func() {
			proj := &hm.Project{BaseDir: Samples()}
			Expect(proj.Load("dep-undefined.hmake")).ShouldNot(BeNil())
			Expect(proj.Name).To(Equal("dep-undefined"))
			Expect(proj.Resolve()).Should(Succeed())
			Expect(proj.Finalize()).
				Should(MatchError(ContainSubstring("not defined")))
		})

		It("duplicated target names", func() {
			proj := &hm.Project{BaseDir: Samples()}
			Expect(proj.Load("dup-target.hmake")).ShouldNot(BeNil())
			Expect(proj.Name).To(Equal("dup-target"))
			Expect(proj.Resolve()).
				Should(MatchError(ContainSubstring("duplicated target")))
		})

		It("returns sorted target names", func() {
			proj := LoadProject(Samples(), "target-names.hmake")
			Expect(proj.TargetNames()).To(Equal([]string{"t0", "t1", "t2", "t3"}))
		})

		It("includes", func() {
			proj := LoadProject(Samples(), "includes.hmake")
			Expect(proj.Files).Should(HaveLen(6))
			Expect(proj.TargetNames()).To(Equal([]string{
				"deep", "inc-a", "inc1", "inc2", "nested",
			}))
		})

		It("includes non-exist", func() {
			proj := &hm.Project{BaseDir: Samples()}
			Expect(proj.Load("include-nonexist.hmake")).ShouldNot(BeNil())
			Expect(proj.Resolve()).ShouldNot(Succeed())
			Expect(proj.Files).Should(HaveLen(2))
		})

		It("retreives settings", func() {
			proj, err := hm.LoadProjectFrom(Fixtures(
				"project0", "subproject", "subdir", "subdir2"))
			Expect(err).Should(Succeed())
			var v struct{}
			Expect(proj.GetSettings(&v)).Should(Succeed())

			proj = &hm.Project{BaseDir: Samples()}
			Expect(proj.Load("includes.hmake")).ShouldNot(BeNil())
			Expect(proj.Resolve()).Should(Succeed())
			Expect(proj.Finalize()).Should(Succeed())
			Expect(proj.MasterFile.Settings).NotTo(BeEmpty())

			var set testSetting
			Expect(proj.GetSettings(&set)).Should(Succeed())
			Expect(set.TopLevel).To(Equal("includes"))
			Expect(set.TopLevel1).To(Equal("inc-a"))
			Expect(set.Dict.Key).To(Equal("inc-a"))
			Expect(set.Dict.Key1).To(Equal("inc-a"))
		})
	})

	Describe("Target", func() {
		It("gets settings and ext", func() {
			proj, err := hm.LoadProjectFrom(Fixtures("project0", "subproject"))
			Expect(err).Should(Succeed())
			Expect(proj.Targets).NotTo(BeEmpty())
			t := proj.Targets["t0"]
			Expect(t).NotTo(BeNil())
			Expect(t.Source).To(Equal("subproject/subproj.hmake"))

			set := &testSetting{}
			Expect(t.GetExt(set)).Should(Succeed())
			Expect(set.TopLevel).To(BeEmpty())
			Expect(set.TopLevel1).To(Equal("t0"))
			set = &testSetting{}
			Expect(t.GetSetting("t0", set)).Should(Succeed())
			Expect(set.TopLevel).To(Equal("project0"))
			Expect(set.TopLevel1).To(Equal("subproj"))
			set = &testSetting{}
			Expect(t.GetSettingWithExt("t0", set)).Should(Succeed())
			Expect(set.TopLevel).To(Equal("project0"))
			Expect(set.TopLevel1).To(Equal("t0"))
		})

		Describe("WatchList", func() {
			It("builds watch list", func() {
				proj, err := hm.LoadProjectFrom(Fixtures("project0", "subproject"))
				Expect(err).Should(Succeed())
				Expect(proj.Targets).NotTo(BeEmpty())
				t := proj.Targets["t0"]
				Expect(t).NotTo(BeNil())

				wl := t.BuildWatchList()
				Expect(wl.IsEmpty()).To(BeFalse())
				Expect(wl).To(HaveLen(2))
				strs := strings.Split(wl.String(), "\n")
				Expect(strs).To(HaveLen(3))
				Expect(strs[0]).To(HavePrefix("subproject/subdir/HyperMake"))
				Expect(strs[1]).To(HavePrefix("subproject/subdir/subdir2/somefile"))
				Expect(strs[2]).To(BeEmpty())
				Expect(wl.Digest()).NotTo(BeEmpty())
			})
		})
	})

	Describe("ExecPlan", func() {

	})
})
