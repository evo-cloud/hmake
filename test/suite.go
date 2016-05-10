package test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

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

		It("merges settings from flat map", func() {
			proj, err := hm.LoadProjectFrom(Fixtures("project0"))
			Expect(err).Should(Succeed())

			v := &testSetting{}
			Expect(proj.GetSettingsIn("t0", v)).Should(Succeed())
			Expect(v.TopLevel).To(Equal("project0"))

			err = proj.MergeSettingsFlat(map[string]interface{}{
				"t0": map[string]interface{}{
					"dict": map[string]interface{}{
						"key": "value",
					},
				},
			})
			Expect(err).Should(Succeed())
			v = &testSetting{}
			Expect(proj.GetSettingsIn("t0", v)).Should(Succeed())
			Expect(v.TopLevel).To(Equal("project0"))
			Expect(v.Dict.Key).To(Equal("value"))
			err = proj.MergeSettingsFlat(map[string]interface{}{
				"t0.dict.key": "value1",
			})
			Expect(err).Should(Succeed())
			Expect(proj.GetSettingsIn("t0", v)).Should(Succeed())
			Expect(v.Dict.Key).To(Equal("value1"))

			err = proj.MergeSettingsFlat(map[string]interface{}{
				"t1.dict.key": "valueX",
			})
			Expect(err).Should(Succeed())
			Expect(proj.GetSettingsIn("t1", v)).Should(Succeed())
			Expect(v.Dict.Key).To(Equal("valueX"))
		})

		It("loads rcfiles", func() {
			proj, err := hm.LoadProjectFrom(Fixtures("project2"))
			Expect(err).Should(Succeed())
			Expect(proj.LoadRcFiles()).Should(Succeed())
			set := &testSetting{}
			Expect(proj.GetSettings(set)).Should(Succeed())
			Expect(set.TopLevel).To(Equal("value1"))

			proj, err = hm.LoadProjectFrom(Fixtures("project2", "subdir"))
			Expect(err).Should(Succeed())
			Expect(proj.LoadRcFiles()).Should(Succeed())
			Expect(proj.GetSettings(set)).Should(Succeed())
			Expect(set.TopLevel).To(Equal("value2"))
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
			Expect(t.GetSettings("t0", set)).Should(Succeed())
			Expect(set.TopLevel).To(Equal("project0"))
			Expect(set.TopLevel1).To(Equal("subproj"))
			set = &testSetting{}
			Expect(t.GetSettingsWithExt("t0", set)).Should(Succeed())
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
		BeforeEach(func() {
			os.RemoveAll(Fixtures("project1", hm.WorkFolder))
		})

		It("generates env", func() {
			proj, err := hm.LoadProjectFrom(Fixtures("project0", "subproject"))
			Expect(err).Should(Succeed())
			plan := proj.Plan()
			Expect(plan.WorkPath).To(Equal(Fixtures("project0", hm.WorkFolder)))
			Expect(plan.Project).To(Equal(proj))
			Expect(plan.Env["HMAKE_PROJECT_DIR"]).To(Equal(Fixtures("project0")))
			Expect(plan.Env["HMAKE_PROJECT_NAME"]).To(Equal("project0"))
			Expect(plan.Env["HMAKE_PROJECT_FILE"]).To(Equal(hm.RootFile))
			Expect(plan.Env["HMAKE_WORK_DIR"]).To(Equal(Fixtures("project0", hm.WorkFolder)))
			Expect(plan.Env["HMAKE_LAUNCH_PATH"]).To(Equal("subproject"))
			Expect(plan.Env["HMAKE_OS"]).To(Equal(runtime.GOOS))
			Expect(plan.Env["HMAKE_ARCH"]).To(Equal(runtime.GOARCH))
		})

		execProject := func(project string, targets ...string) (plan *hm.ExecPlan, execOrder []string) {
			proj, err := hm.LoadProjectFrom(Fixtures(project))
			Expect(err).Should(Succeed())
			plan = proj.Plan()
			plan.DebugLog = true
			execCh := make(chan string)
			plan.RunnerFactory = func(task *hm.Task) (hm.Runner, error) {
				runner, err := task.Runner()
				Expect(err).Should(Succeed())
				return func(task *hm.Task) (hm.TaskResult, error) {
					execCh <- task.Name()
					return runner(task)
				}, nil
			}
			for _, t := range targets {
				if t == "-R" {
					plan.RebuildAll = true
				} else if strings.HasPrefix(t, "-r:") {
					plan.Rebuild(t[3:])
				} else if strings.HasPrefix(t, "-s:") {
					plan.Skip(t[3:])
				} else {
					plan.Require(t)
				}
			}
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				for {
					name, ok := <-execCh
					if !ok {
						break
					}
					execOrder = append(execOrder, name)
				}
				wg.Done()
			}()
			Expect(plan.Execute()).Should(Succeed())
			close(execCh)
			wg.Wait()
			return
		}

		It("executes tasks in right order", func() {
			plan, execOrder := execProject("project1", "all")
			names := plan.Project.TargetNames()
			Expect(execOrder).Should(HaveLen(len(names)))
			startNum := 0
			for i := 0; i+1 < len(execOrder); i++ {
				name := execOrder[i]
				num, err := strconv.Atoi(name[1:2])
				Expect(err).Should(Succeed())
				Expect(num).ShouldNot(BeNumerically("<", startNum))
				if num > startNum {
					startNum = num
				}
			}
		})

		It("skips tasks without file changes", func() {
			os.RemoveAll(Fixtures("project1", "touch.log"))
			_, execOrder0 := execProject("project1", "all")
			_, execOrder1 := execProject("project1", "all")
			Expect(execOrder1).Should(HaveLen(len(execOrder0) - 3))
			for i := 0; i < len(execOrder1); i++ {
				name := execOrder1[i]
				Expect(name).ShouldNot(Equal("t0"))
				Expect(name).ShouldNot(HavePrefix("t1"))
			}
			Expect(ioutil.WriteFile(
				Fixtures("project1", "touch.log"), []byte("touch"), 0644)).
				Should(Succeed())
			_, execOrder2 := execProject("project1", "all")
			Expect(execOrder2).Should(HaveLen(len(execOrder0)))
		})

		It("rebuilds task when explicitly specified", func() {
			os.RemoveAll(Fixtures("project1", "touch.log"))
			_, execOrder0 := execProject("project1", "all")
			_, execOrder1 := execProject("project1", "all")
			Expect(execOrder1).Should(HaveLen(len(execOrder0) - 3))
			_, execOrder2 := execProject("project1", "-r:t0", "all")
			Expect(execOrder2).Should(HaveLen(len(execOrder0)))
			_, execOrder3 := execProject("project1", "-R", "all")
			Expect(execOrder3).Should(HaveLen(len(execOrder0)))
		})

		It("skips task when explicitly specified", func() {
			os.RemoveAll(Fixtures("project1", "touch.log"))
			_, execOrder0 := execProject("project1", "all")
			_, execOrder1 := execProject("project1", "all", "-s:t2")
			Expect(execOrder1).Should(HaveLen(len(execOrder0) - 4))
		})

		It("generates summary file", func() {
			plan, _ := execProject("project1", "t0")
			data, err := ioutil.ReadFile(plan.Project.SummaryFile())
			Expect(err).Should(Succeed())
			var summary []map[string]interface{}
			Expect(json.Unmarshal(data, &summary)).Should(Succeed())
			Expect(summary).To(HaveLen(1))
			Expect(summary[0]["target"]).To(Equal("t0"))
			Expect(summary[0]["result"]).To(Equal("Success"))
		})

		It("emits event and task failure", func() {
			os.RemoveAll(Fixtures("project1", "touch.log"))
			execProject("project1", "all")

			taskFails := map[string]bool{
				"t2": true,
			}

			taskResults := make(map[string]hm.TaskResult)

			proj, err := hm.LoadProjectFrom(Fixtures("project1"))
			Expect(err).Should(Succeed())
			plan := proj.Plan()
			plan.RunnerFactory = func(task *hm.Task) (hm.Runner, error) {
				return func(task *hm.Task) (hm.TaskResult, error) {
					if _, exists := taskFails[task.Name()]; exists {
						return hm.Failure, nil
					}
					return hm.Success, nil
				}, nil
			}
			plan.Require("all")
			plan.OnEvent(func(event interface{}) {
				switch evt := event.(type) {
				case *hm.EvtTaskFinish:
					taskResults[evt.Task.Name()] = evt.Task.Result
				}
			})
			Expect(plan.Execute()).ShouldNot(Succeed())
			Expect(taskResults).Should(HaveLen(4))
			Expect(taskResults["t0"]).To(Equal(hm.Skipped))
			Expect(taskResults["t1.0"]).To(Equal(hm.Skipped))
			Expect(taskResults["t1.1"]).To(Equal(hm.Skipped))
			Expect(taskResults["t2"]).To(Equal(hm.Failure))
			Expect(plan.Tasks["t0"].Duration()).To(BeZero())
		})

		It("converts states into strings", func() {
			Expect(hm.Unknown.String()).To(BeEmpty())
			Expect(hm.Success.String()).To(Equal("Success"))
			Expect(hm.Failure.String()).To(Equal("Failure"))
			Expect(hm.Skipped.String()).To(Equal("Skipped"))
			Expect(hm.Waiting.String()).To(Equal("Waiting"))
			Expect(hm.Queued.String()).To(Equal("Queued"))
			Expect(hm.Running.String()).To(Equal("Running"))
			Expect(hm.Finished.String()).To(Equal("Finished"))
		})

		It("provides default script/log paths", func() {
			proj, err := hm.LoadProjectFrom(Fixtures("project1"))
			Expect(err).Should(Succeed())
			plan := proj.Plan()
			plan.Require("all", "t2", "t3.0")
			os.MkdirAll(plan.WorkPath, 0755)
			task := plan.Tasks["all"]
			Expect(task.ScriptFile()).To(Equal(Fixtures("project1", hm.WorkFolder, "all.script")))
			Expect(task.LogFile()).To(Equal(Fixtures("project1", hm.WorkFolder, "all.log")))
			task = plan.Tasks["t3.0"]
			script, err := task.BuildScriptFile()
			Expect(err).Should(Succeed())
			Expect(script).To(Equal("#!/usr/bin/interpreter"))
			fileContent, err := ioutil.ReadFile(Fixtures("project1", hm.WorkFolder, "t3.0.script"))
			Expect(err).Should(Succeed())
			Expect(string(fileContent)).To(Equal(script))
			task = plan.Tasks["t2"]
			script, err = task.BuildScriptFile()
			Expect(err).Should(Succeed())
			Expect(script).To(HavePrefix("#!/bin/sh\n"))
			Expect(task.ExecScript()).Should(Succeed())
			fileContent, err = ioutil.ReadFile(Fixtures("project1", hm.WorkFolder, "t2.log"))
			Expect(err).Should(Succeed())
			Expect(string(fileContent)).To(Equal("hello"))
		})
	})
})
