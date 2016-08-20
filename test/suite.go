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
	sh "github.com/evo-cloud/hmake/shell"
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

// LoadFixtureProject loads project from fixtures
func LoadFixtureProject(fixtures ...string) *hm.Project {
	proj, err := hm.LoadProjectFrom(Fixtures(fixtures...), hm.RootFile)
	Expect(err).Should(Succeed())
	return proj
}

type testRunner struct {
	task *hm.Task
	run  func(task *hm.Task) (hm.TaskResult, error)
}

type testRunnerExt struct {
	Touch string `json:"touch"`
	Do    string `json:"do"`
}

func (r *testRunner) Run(sigCh <-chan os.Signal) (hm.TaskResult, error) {
	var ext testRunnerExt
	r.task.Target.GetExt(&ext)
	r.task.Result = hm.Success
	if ext.Touch != "" {
		if f, err := os.Create(r.task.WorkingDir(ext.Touch)); err != nil {
			r.task.Result = hm.Failure
			r.task.Error = err
		} else {
			f.Close()
		}
	}
	if r.run != nil {
		return r.run(r.task)
	}
	return r.task.Result, r.task.Error
}

func (r *testRunner) Signature() string {
	var ext testRunnerExt
	r.task.Target.GetExt(&ext)
	return ext.Do
}

func (r *testRunner) ValidateArtifacts() bool {
	return true
}

type testSetting struct {
	TopLevel  string `json:"toplevel"`
	TopLevel1 string `json:"toplevel1"`
	Local1    string `json:"local1"`
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
	// register shell driver for testing
	hm.RegisterExecDriver(sh.ExecDriverName, sh.Factory)
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
			proj := LoadFixtureProject("project0", "subproject", "subdir", "subdir2")
			Expect(proj.Name).To(Equal("subdir"))
			proj = LoadFixtureProject("project0", "subproject", "subdir")
			Expect(proj.Name).To(Equal("subdir"))
			proj = LoadFixtureProject("project0", "subproject")
			Expect(proj.Name).To(Equal("project0"))
			_, err := hm.LoadProjectFrom("/", hm.RootFile)
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

		It("matches target names", func() {
			proj := LoadFixtureProject("project1")
			names, err := proj.TargetNamesMatch("t?")
			Expect(err).Should(Succeed())
			Expect(names).To(Equal([]string{"t0", "t2"}))
			names, err = proj.TargetNamesMatch("t3*")
			Expect(err).Should(Succeed())
			Expect(names).To(Equal([]string{"t3.0", "t3.1", "t3.2", "t3.3"}))
			names, err = proj.TargetNamesMatch(`/[^\.]+\.[^\.]+/`)
			Expect(err).Should(Succeed())
			Expect(names).To(Equal([]string{"t1.0", "t1.1", "t3.0", "t3.1", "t3.2", "t3.3"}))
			_, err = proj.TargetNamesMatch("t[")
			Expect(err).To(HaveOccurred())
			_, err = proj.TargetNamesMatch("/.")
			Expect(err).To(HaveOccurred())
			_, err = proj.TargetNamesMatch("/[./")
			Expect(err).To(HaveOccurred())
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
			proj := LoadFixtureProject("project0", "subproject", "subdir", "subdir2")
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
			proj := LoadFixtureProject("project0")

			v := &testSetting{}
			Expect(proj.GetSettingsIn("t0", v)).Should(Succeed())
			Expect(v.TopLevel).To(Equal("project0"))

			err := proj.MergeSettingsFlat(map[string]interface{}{
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
			proj := LoadFixtureProject("project2")
			Expect(proj.LoadRcFiles()).Should(Succeed())
			set := &testSetting{}
			Expect(proj.GetSettings(set)).Should(Succeed())
			Expect(set.TopLevel).To(Equal("value1"))

			proj = LoadFixtureProject("project2", "subdir")
			Expect(proj.LoadRcFiles()).Should(Succeed())
			Expect(proj.GetSettings(set)).Should(Succeed())
			Expect(set.TopLevel).To(Equal("value2"))
		})
	})

	Describe("Target", func() {
		It("gets settings and ext", func() {
			proj := LoadFixtureProject("project0", "subproject")
			Expect(proj.Targets).NotTo(BeEmpty())
			t := proj.Targets["t0"]
			Expect(t).NotTo(BeNil())
			Expect(t.File.Source).To(Equal("subproject/subproj.hmake"))
			Expect(t.WorkingDir()).To(Equal("subproject/subdir"))

			set := &testSetting{}
			Expect(t.GetExt(set)).Should(Succeed())
			Expect(set.TopLevel).To(BeEmpty())
			Expect(set.TopLevel1).To(Equal("t0"))
			set = &testSetting{}
			Expect(t.GetSettings("t0", set)).Should(Succeed())
			Expect(set.TopLevel).To(Equal("project0"))
			Expect(set.TopLevel1).To(Equal("subproj"))
			Expect(set.Local1).To(Equal("subproj"))
			set = &testSetting{}
			Expect(t.GetSettingsWithExt("t0", set)).Should(Succeed())
			Expect(set.TopLevel).To(Equal("project0"))
			Expect(set.TopLevel1).To(Equal("t0"))
			Expect(set.Local1).To(Equal("subproj"))

			t = proj.Targets["t"]
			set = &testSetting{}
			Expect(t.GetSettings("t0", set)).Should(Succeed())
			Expect(set.TopLevel).To(Equal("project0"))
			Expect(set.TopLevel1).To(Equal("subproj"))
			Expect(set.Local1).To(BeEmpty())
		})

		Describe("WatchList", func() {
			It("builds watch list", func() {
				proj := LoadFixtureProject("project0", "subproject")
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
			})
		})
	})

	Describe("ExecPlan", func() {
		BeforeEach(func() {
			os.RemoveAll(Fixtures("project1", hm.WorkFolder))
		})

		It("generates env", func() {
			proj := LoadFixtureProject("project0", "subproject")
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
			if len(targets) > 0 && strings.HasPrefix(targets[0], "-f:") {
				projectFile := targets[0][3:]
				targets = targets[1:]
				proj, err := hm.LoadProjectFrom(Fixtures(project), projectFile)
				Expect(err).Should(Succeed())
				plan = proj.Plan()
			} else {
				plan = LoadFixtureProject(project).Plan()
			}
			plan.DebugLog = true
			execCh := make(chan string)
			plan.RunnerFactory = func(task *hm.Task) (hm.Runner, error) {
				return &testRunner{
					task: task,
					run: func(task *hm.Task) (hm.TaskResult, error) {
						execCh <- task.Name()
						return hm.Success, nil
					},
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
			Expect(plan.Execute(nil)).Should(Succeed())
			close(execCh)
			wg.Wait()
			return
		}

		It("executes tasks in right order", func() {
			os.RemoveAll(Fixtures("exec-order", hm.WorkFolder))
			plan, execOrder := execProject("exec-order", "all")
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
			os.Remove(Fixtures("project1", "touch.log"))
			_, execOrder0 := execProject("project1", "all")
			_, execOrder1 := execProject("project1", "all")
			Expect(execOrder1).Should(BeEmpty())
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

		It("rebuilds task with task changed", func() {
			os.RemoveAll(Fixtures("task-change", hm.WorkFolder))
			plan, execOrder0 := execProject("task-change", "all")
			Expect(plan.Project.MasterFile.Source).Should(Equal("HyperMake"))
			Expect(execOrder0).Should(HaveLen(2))
			plan, execOrder1 := execProject("task-change", "all")
			Expect(plan.Project.MasterFile.Source).Should(Equal("HyperMake"))
			Expect(execOrder1).Should(BeEmpty())
			plan, execOrder2 := execProject("task-change", "-f:HyperMake.changed", "all")
			Expect(plan.Project.MasterFile.Source).Should(Equal("HyperMake.changed"))
			Expect(execOrder2).Should(HaveLen(1))
		})

		It("rebuilds task when explicitly specified", func() {
			os.Remove(Fixtures("project1", "touch.log"))
			_, execOrder0 := execProject("project1", "all")
			_, execOrder1 := execProject("project1", "all")
			Expect(execOrder1).Should(BeEmpty())
			_, execOrder2 := execProject("project1", "-r:t0", "all")
			Expect(execOrder2).Should(HaveLen(len(execOrder0)))
			_, execOrder3 := execProject("project1", "-R", "all")
			Expect(execOrder3).Should(HaveLen(len(execOrder0)))
		})

		It("rebuilds task when artifacts missing", func() {
			os.Remove(Fixtures("artifacts", "test.log"))
			_, execOrder0 := execProject("artifacts", "t0")
			Expect(execOrder0).Should(HaveLen(1))
			_, execOrder1 := execProject("artifacts", "t0")
			Expect(execOrder1).Should(BeEmpty())
			os.Remove(Fixtures("artifacts", "test.log"))
			_, execOrder2 := execProject("artifacts", "t0")
			Expect(execOrder2).Should(HaveLen(1))
		})

		It("fail the task if artifacts are incomplete", func() {
			os.Remove(Fixtures("artifacts", "test2.log"))
			plan := LoadFixtureProject("artifacts").Plan()
			var taskErr error
			plan.RunnerFactory = func(task *hm.Task) (hm.Runner, error) {
				return &testRunner{
					task: task,
					run: func(task *hm.Task) (hm.TaskResult, error) {
						return hm.Success, nil
					},
				}, nil
			}
			plan.Require("t2")
			plan.OnEvent(func(event interface{}) {
				switch evt := event.(type) {
				case *hm.EvtTaskFinish:
					if evt.Task.Result == hm.Failure {
						taskErr = evt.Task.Error
					}
				}
			})
			Expect(plan.Execute(nil)).ShouldNot(Succeed())
			Expect(taskErr).ShouldNot(Succeed())
		})

		It("skips task when explicitly specified", func() {
			os.Remove(Fixtures("project1", "touch.log"))
			_, execOrder0 := execProject("project1", "all")
			_, execOrder1 := execProject("project1", "all", "-R", "-s:t2")
			Expect(execOrder1).Should(HaveLen(len(execOrder0) - 1))
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
			os.Remove(Fixtures("project1", "touch.log"))
			execProject("project1", "all")

			taskFails := map[string]bool{
				"t2": true,
			}

			taskResults := make(map[string]hm.TaskResult)

			plan := LoadFixtureProject("project1").Plan()
			plan.RunnerFactory = func(task *hm.Task) (hm.Runner, error) {
				return &testRunner{
					task: task,
					run: func(task *hm.Task) (hm.TaskResult, error) {
						if _, exists := taskFails[task.Name()]; exists {
							return hm.Failure, nil
						}
						return hm.Success, nil
					},
				}, nil
			}
			plan.Require("all")
			plan.RebuildAll = true
			plan.OnEvent(func(event interface{}) {
				switch evt := event.(type) {
				case *hm.EvtTaskFinish:
					taskResults[evt.Task.Name()] = evt.Task.Result
				}
			})
			Expect(plan.Execute(nil)).ShouldNot(Succeed())
			Expect(taskResults["t2"]).To(Equal(hm.Failure))
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
			plan := LoadFixtureProject("project1").Plan()
			plan.Require("all", "t2", "t3.0")
			os.MkdirAll(plan.WorkPath, 0755)
			task := plan.Tasks["all"]
			Expect(sh.ScriptFile(task)).To(Equal(Fixtures("project1", hm.WorkFolder, "all.script")))
			Expect(sh.LogFile(task)).To(Equal(Fixtures("project1", hm.WorkFolder, "all.log")))
			task = plan.Tasks["t3.0"]
			script, err := sh.BuildScriptFile(task)
			Expect(err).Should(Succeed())
			Expect(script).To(Equal("#!/usr/bin/interpreter"))
			fileContent, err := ioutil.ReadFile(Fixtures("project1", hm.WorkFolder, "t3.0.script"))
			Expect(err).Should(Succeed())
			Expect(string(fileContent)).To(Equal(script))
			task = plan.Tasks["t2"]
			script, err = sh.BuildScriptFile(task)
			Expect(err).Should(Succeed())
			Expect(script).To(HavePrefix("#!/bin/sh\n"))
			Expect(sh.ExecScript(task).Run(nil)).Should(Succeed())
			fileContent, err = ioutil.ReadFile(Fixtures("project1", hm.WorkFolder, "t2.log"))
			Expect(err).Should(Succeed())
			Expect(string(fileContent)).To(Equal("hello"))
		})

		It("terminates the running targets", func() {
			plan := LoadFixtureProject("project-abort").Plan()
			plan.Require("abort0")
			os.MkdirAll(plan.WorkPath, 0755)

			t := plan.Tasks["abort0"]
			Expect(t).NotTo(BeNil())
			ch := make(chan os.Signal, 2)

			taskResults := make(map[string]hm.TaskResult)
			plan.OnEvent(func(event interface{}) {
				switch evt := event.(type) {
				case *hm.EvtTaskStart:
					if evt.Task.Name() == "abort0" {
						go func() {
							ch <- os.Interrupt
						}()
					}
				case *hm.EvtTaskFinish:
					taskResults[evt.Task.Name()] = evt.Task.Result
				}
			})
			Expect(plan.Execute(ch)).ShouldNot(Succeed())
			Expect(taskResults["abort0"]).To(Equal(hm.Failure))
		})

		It("skips transit targets when all dependencies are skipped", func() {
			proj := LoadFixtureProject("skip-transit-targets")
			plan := proj.Plan()
			plan.Require("all")
			os.MkdirAll(plan.WorkPath, 0755)
			Expect(plan.Execute(nil)).Should(Succeed())
			// run to generate success marks
			// now t0 t1 should be skipped
			plan = proj.Plan()
			plan.Require("all")
			taskResults := make(map[string]hm.TaskResult)
			handler := func(event interface{}) {
				switch evt := event.(type) {
				case *hm.EvtTaskFinish:
					taskResults[evt.Task.Name()] = evt.Task.Result
				}
			}
			plan.OnEvent(handler)
			Expect(plan.Execute(nil)).Should(Succeed())
			Expect(taskResults["t0"]).To(Equal(hm.Skipped))
			Expect(taskResults["t1"]).To(Equal(hm.Skipped))
			Expect(taskResults["all"]).To(Equal(hm.Skipped))
			Expect(plan.Tasks["all"].Duration()).Should(BeZero())

			// run again with dependencies rebuilt
			plan = proj.Plan()
			plan.Require("all")
			plan.Rebuild("t0")
			taskResults = make(map[string]hm.TaskResult)
			plan.OnEvent(handler)
			Expect(plan.Execute(nil)).Should(Succeed())
			Expect(taskResults["t0"]).To(Equal(hm.Success))
			Expect(taskResults["t1"]).To(Equal(hm.Skipped))
			Expect(taskResults["all"]).To(Equal(hm.Success))
		})

		It("always build target with property always set to true", func() {
			proj := LoadFixtureProject("always-target")
			plan := proj.Plan()
			plan.Require("all")
			os.MkdirAll(plan.WorkPath, 0755)
			Expect(plan.Execute(nil)).Should(Succeed())
			// run to generate success marks
			// now t0 should be skipped, all should be success
			plan = proj.Plan()
			plan.Require("all")
			taskResults := make(map[string]hm.TaskResult)
			handler := func(event interface{}) {
				switch evt := event.(type) {
				case *hm.EvtTaskFinish:
					taskResults[evt.Task.Name()] = evt.Task.Result
				}
			}
			plan.OnEvent(handler)
			Expect(plan.Execute(nil)).Should(Succeed())
			Expect(taskResults["t0"]).To(Equal(hm.Skipped))
			Expect(taskResults["t0a"]).To(Equal(hm.Skipped))
			Expect(taskResults["t1"]).To(Equal(hm.Success))
			Expect(taskResults["t2"]).To(Equal(hm.Success))
			Expect(taskResults["all"]).To(Equal(hm.Success))

			// run again with explicity skip
			plan = proj.Plan()
			plan.Skip("t1")
			plan.Require("all")
			taskResults = make(map[string]hm.TaskResult)
			plan.OnEvent(handler)
			Expect(plan.Execute(nil)).Should(Succeed())
			Expect(taskResults["t0"]).To(Equal(hm.Skipped))
			Expect(taskResults["t0a"]).To(Equal(hm.Skipped))
			Expect(taskResults["t1"]).To(Equal(hm.Skipped))
			Expect(taskResults["t2"]).To(Equal(hm.Skipped))
			Expect(taskResults["all"]).To(Equal(hm.Skipped))
		})
	})
})
