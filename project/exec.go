package project

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/easeway/langx.go/errors"
)

// ExecPlan describes the plan for execution
type ExecPlan struct {
	// Project is the wrapped project
	Project *Project
	// Tasks are the tasks need execution
	Tasks map[string]*Task
	// Env is pre-defined environment variables
	Env map[string]string
	// WorkPath is full path to hmake state dir ProjectDir/.hmake
	WorkPath string
	// MaxConcurrency defines maximum number of tasks being executed in parallel
	// if it's 0, the number of CPU cores are counted
	MaxConcurrency int
	// RebuildAll force rebuild everything regardless of success mark
	RebuildAll bool
	// RebuildTargets specify targets to rebuild regardless of success mark
	RebuildTargets map[string]bool
	// SkippedTargets specify targets to be marked as Skipped
	SkippedTargets map[string]bool
	// RequiredTargets are names of targets explicitly required
	RequiredTargets []string
	// RunnerFactory specifies the custom runner factory
	RunnerFactory RunnerFactory
	// DebugLog enables logging debug info into .hmake/hmake.debug.log
	DebugLog bool
	// Dryrun will skip the actual execution of target, just return success
	DryRun bool
	// WaitingTasks are tasks in waiting state
	WaitingTasks map[string]*Task
	// QueuedTasks are tasks in Queued state
	QueuedTasks []*Task
	// RunningTasks are tasks in Running state
	RunningTasks map[string]*Task
	// FinishedTasks are tasks in finished state
	FinishedTasks []*Task
	// EventHandler handles the events during execution
	EventHandler EventHandler
	// Summary is the report of all executed targets
	Summary []map[string]interface{}

	finishCh chan *Task
	logger   *log.Logger
}

// EventHandler receives event notifications during execution of plan
type EventHandler func(event interface{})

// EvtTaskStart is emitted before task gets run
type EvtTaskStart struct {
	Task *Task
}

// EvtTaskFinish is emitted when task finishes
type EvtTaskFinish struct {
	Task *Task
}

// EvtTaskActivated is emitted when task is queued
type EvtTaskActivated struct {
	Task *Task
}

// EvtTaskOutput is emitted when output is received
type EvtTaskOutput struct {
	Task   *Task
	Output []byte
}

// Task is the execution state of a target
type Task struct {
	// Plan is ExecPlan owns the task
	Plan *ExecPlan
	// Target is wrapped target
	Target *Target
	// Depends is the tasks being depended on
	// The task is activated when depends is empty
	Depends map[string]*Task
	// State indicates task state
	State TaskState
	// Result indicates the result of the task
	Result TaskResult
	// Error represents any error happened during execution
	Error error
	// StartTime
	StartTime time.Time
	// FinishTime
	FinishTime time.Time

	alwaysBuild   bool
	currentDigest string
}

// TaskResult indicates the result of task execution
type TaskResult int

// Task results
const (
	Unknown TaskResult = iota
	Success
	Failure
	Skipped
)

func (r TaskResult) String() string {
	switch r {
	case Unknown:
		return ""
	case Success:
		return "Success"
	case Failure:
		return "Failure"
	case Skipped:
		return "Skipped"
	}
	panic("invalid TaskResult " + strconv.Itoa(int(r)))
}

// TaskState indicates the state of task
type TaskState int

// Task states
const (
	Waiting TaskState = iota
	Queued
	Running
	Finished
)

func (r TaskState) String() string {
	switch r {
	case Waiting:
		return "Waiting"
	case Queued:
		return "Queued"
	case Running:
		return "Running"
	case Finished:
		return "Finished"
	}
	panic("invalid TaskState " + strconv.Itoa(int(r)))
}

// Runner is the handler execute a target
type Runner func(*Task) (TaskResult, error)

// RunnerFactory creates a runner from a task
type RunnerFactory func(*Task) (Runner, error)

const (
	// SettingExecDriver is the property name of exec-driver
	SettingExecDriver = "exec-driver"
)

var (
	// DefaultExecDriver specify the default exec-driver to use
	DefaultExecDriver string

	runners = make(map[string]Runner)
)

// RegisterExecDriver registers a runner
func RegisterExecDriver(name string, runner Runner) {
	runners[name] = runner
}

// NewExecPlan creates an ExecPlan for a Project
func NewExecPlan(project *Project) *ExecPlan {
	plan := &ExecPlan{
		Project:        project,
		RebuildTargets: make(map[string]bool),
		SkippedTargets: make(map[string]bool),
		Tasks:          make(map[string]*Task),
		Env:            make(map[string]string),
		WorkPath:       filepath.Join(project.BaseDir, WorkFolder),
		WaitingTasks:   make(map[string]*Task),

		logger: log.New(ioutil.Discard, "", log.Ltime),
	}
	plan.Env["HMAKE_PROJECT_NAME"] = project.Name
	plan.Env["HMAKE_PROJECT_DIR"] = project.BaseDir
	plan.Env["HMAKE_PROJECT_FILE"] = project.MasterFile.Source
	plan.Env["HMAKE_WORK_DIR"] = plan.WorkPath
	plan.Env["HMAKE_LAUNCH_PATH"] = project.LaunchPath
	plan.Env["HMAKE_OS"] = runtime.GOOS
	plan.Env["HMAKE_ARCH"] = runtime.GOARCH
	return plan
}

// OnEvent subscribes the events
func (p *ExecPlan) OnEvent(handler EventHandler) *ExecPlan {
	p.EventHandler = handler
	return p
}

// Rebuild specify specific targets to be rebuilt
func (p *ExecPlan) Rebuild(targets ...string) *ExecPlan {
	for _, target := range targets {
		p.RebuildTargets[target] = true
	}
	return p
}

// Skip specify the targets to be skipped
func (p *ExecPlan) Skip(targets ...string) *ExecPlan {
	for _, target := range targets {
		p.SkippedTargets[target] = true
	}
	return p
}

// Require adds targets to be executed
func (p *ExecPlan) Require(targets ...string) error {
	if p.Tasks == nil {
		p.Tasks = make(map[string]*Task)
	}
	if p.WaitingTasks == nil {
		p.WaitingTasks = make(map[string]*Task)
	}
	errs := &errors.AggregatedError{}
	for _, name := range targets {
		t := p.Project.Targets[name]
		if t == nil {
			errs.Add(fmt.Errorf("target %s not defined", name))
		} else if _, added := p.AddTarget(t); added {
			p.RequiredTargets = append(p.RequiredTargets, name)
		}
	}
	return errs.Aggregate()
}

// AddTarget adds a target into execution plan
func (p *ExecPlan) AddTarget(t *Target) (*Task, bool) {
	task, exists := p.Tasks[t.Name]
	if !exists {
		task = NewTask(p, t)
		p.Tasks[t.Name] = task
		for name, dep := range t.Depends {
			task.Depends[name], _ = p.AddTarget(dep)
		}
		if task.IsActivated() {
			task.State = Queued
			p.QueuedTasks = append(p.QueuedTasks, task)
		} else {
			task.State = Waiting
			p.WaitingTasks[t.Name] = task
		}
	}
	return task, !exists
}

// Execute start execution
func (p *ExecPlan) Execute() error {
	p.Env["HMAKE_REQUIRED_TARGETS"] = strings.Join(p.RequiredTargets, " ")

	// DryRun should not make any changes
	if !p.DryRun {
		if err := os.MkdirAll(p.WorkPath, 0755); err != nil {
			return err
		}

		if p.DebugLog {
			f, err := os.OpenFile(p.Project.DebugLogFile(),
				syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC, 0644)
			if err == nil {
				defer f.Close()
				p.logger = log.New(f, "hmake: ", log.Ltime)
			}
		}
	}

	errs := &errors.AggregatedError{}
	p.finishCh = make(chan *Task)
	p.RunningTasks = make(map[string]*Task)

	concurrency := p.MaxConcurrency
	if concurrency == 0 {
		concurrency = runtime.NumCPU()
	}

	p.logger.Printf("RebuildAll = %v\n", p.RebuildAll)
	p.logger.Printf("Rebuild = %v\n", p.RebuildTargets)
	p.logger.Printf("Concurrency = %v\n", concurrency)

	for _, task := range p.QueuedTasks {
		p.logger.Printf("Activate %s\n", task.Name())
		p.emit(&EvtTaskActivated{Task: task})
	}

	for {
		tasks := p.dequeueTasks(concurrency)
		if len(tasks) > 0 {
			runningCount := len(p.RunningTasks)
			for _, task := range tasks {
				p.startTask(task, errs)
			}
			if len(p.RunningTasks) < runningCount+len(tasks) {
				// not all tasks pushed to runningTasks
				// means some tasks are skipped or failed immediately, thus
				// other tasks may be activated, need to dequeue again
				continue
			}
		} else if len(p.RunningTasks) == 0 {
			// nothing to run
			break
		}

		if len(p.RunningTasks) > 0 {
			p.finishTask(<-p.finishCh, errs)
		}
	}

	p.GenerateSummary()

	return errs.Aggregate()
}

// GenerateSummary dumps summary to summary file
func (p *ExecPlan) GenerateSummary() (err error) {
	var sum []map[string]interface{}
	for _, t := range p.FinishedTasks {
		sum = append(sum, t.Summary())
	}
	for _, t := range p.RunningTasks {
		sum = append(sum, t.Summary())
	}
	for _, t := range p.QueuedTasks {
		sum = append(sum, t.Summary())
	}
	for _, t := range p.WaitingTasks {
		sum = append(sum, t.Summary())
	}
	p.Summary = sum
	encoded, _ := json.Marshal(sum)
	p.logger.Printf("Summary\n%s\n", string(encoded))
	if !p.DryRun {
		err = ioutil.WriteFile(p.Project.SummaryFile(), encoded, 0644)
		if err != nil {
			p.logger.Printf("Write summary failed: %v\n", err)
		}
	}
	return err
}

func (p *ExecPlan) dequeueTasks(dequeueCnt int) (tasks []*Task) {
	if dequeueCnt < 0 {
		// unlimited, dequeue all
		dequeueCnt = len(p.QueuedTasks)
	} else {
		// exclude runnings from dequeueCnt
		dequeueCnt -= len(p.RunningTasks)
		// make sure dequeueCnt <= len(queued)
		if l := len(p.QueuedTasks); dequeueCnt > l {
			dequeueCnt = l
		}
	}
	if dequeueCnt > 0 {
		tasks = p.QueuedTasks[0:dequeueCnt]
		p.QueuedTasks = p.QueuedTasks[dequeueCnt:]
	}
	return
}

func (p *ExecPlan) emit(event interface{}) {
	if p.EventHandler != nil {
		p.EventHandler(event)
	}
}

func (p *ExecPlan) startTask(task *Task, errs *errors.AggregatedError) {
	p.logger.Printf("Start %s\n", task.Name())
	task.State = Running
	p.RunningTasks[task.Name()] = task
	task.StartTime = time.Now()
	p.emit(&EvtTaskStart{Task: task})

	skippable := task.CalcSuccessMark()
	if p.SkippedTargets[task.Name()] {
		skippable = true
	} else if p.RebuildAll || p.RebuildTargets[task.Name()] {
		skippable = false
	}

	if skippable {
		task.Result = Skipped
		task.FinishTime = task.StartTime
		p.finishTask(task, errs)
		return
	}

	task.ClearSuccessMark()
	for name := range task.Target.Activates {
		if t := p.Tasks[name]; t != nil {
			t.ClearSuccessMark()
		}
	}

	runner, err := p.runner(task)
	if err != nil {
		task.Error = err
		task.Result = Failure
		p.finishTask(task, errs)
	} else {
		go p.run(task, runner)
	}
}

func (p *ExecPlan) runner(task *Task) (Runner, error) {
	if p.RunnerFactory != nil {
		return p.RunnerFactory(task)
	}
	return task.Runner()
}

func (p *ExecPlan) run(task *Task, runner Runner) {
	if p.DryRun {
		task.Result = Success
	} else {
		task.Result, task.Error = runner(task)
	}
	task.FinishTime = time.Now()
	p.finishCh <- task
}

func (p *ExecPlan) finishTask(task *Task, errs *errors.AggregatedError) {
	if _, exist := p.RunningTasks[task.Name()]; !exist {
		// task is out-of-date, ignored
		p.logger.Printf("OUT-OF-DATE %s Result = %s, Err = %v\n",
			task.Name(), task.Result.String(), task.Error)
		return
	}

	p.logger.Printf("Finish %s Result = %s, Err = %v\n",
		task.Name(), task.Result.String(), task.Error)

	// transit to finished state
	task.State = Finished
	delete(p.RunningTasks, task.Name())
	p.FinishedTasks = append(p.FinishedTasks, task)
	if !p.DryRun {
		err := task.BuildSuccessMark()
		if err != nil {
			p.logger.Printf("IGNORED: %s BuildSuccessMark Error: %v\n",
				task.Name(), err)
		}
	}

	p.emit(&EvtTaskFinish{Task: task})

	errs.Add(task.Error)
	if task.Result == Failure {
		if task.Error == nil {
			errs.Add(task.Target.Errorf("failed"))
		}
		return
	}

	// Activate other tasks on success
	for name := range task.Target.Activates {
		t := p.Tasks[name]
		if t == nil {
			continue
		}
		delete(t.Depends, task.Name())

		if t.IsActivated() && p.WaitingTasks[t.Name()] != nil {
			delete(p.WaitingTasks, t.Name())
			t.State = Queued
			p.QueuedTasks = append(p.QueuedTasks, t)
			p.logger.Printf("Activate %s", t.Name())
			p.emit(&EvtTaskActivated{Task: t})
		}
	}
}

// NewTask creates a task for a target
func NewTask(p *ExecPlan, t *Target) *Task {
	return &Task{Plan: p, Target: t, Depends: make(map[string]*Task)}
}

// Name returns the name of wrapped target
func (t *Task) Name() string {
	return t.Target.Name
}

// Project returns the associated project
func (t *Task) Project() *Project {
	return t.Target.Project
}

// IsActivated indicates the task is ready to run
func (t *Task) IsActivated() bool {
	return len(t.Depends) == 0
}

// Duration is how long the task executed
func (t *Task) Duration() time.Duration {
	return t.FinishTime.Sub(t.StartTime)
}

func timeToString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	encoded, err := t.MarshalText()
	if err != nil {
		return ""
	}
	return string(encoded)
}

// Summary generates summary info of the task
func (t *Task) Summary() map[string]interface{} {
	info := map[string]interface{}{
		"target": t.Name(),
		"state":  t.State.String(),
	}
	if s := timeToString(t.StartTime); s != "" {
		info["start-at"] = s
	}
	if s := timeToString(t.FinishTime); s != "" {
		info["finish-at"] = s
	}
	if t.State == Finished {
		info["result"] = t.Result.String()
		if t.Error != nil {
			info["error"] = t.Error.Error()
		}
	}
	return info
}

// CalcSuccessMark calculates the watchlist digest and
// checks if the task can be skipped
func (t *Task) CalcSuccessMark() bool {
	wl := t.Target.BuildWatchList()
	t.Plan.logger.Printf("%s WatchList:\n%s", t.Name(), wl.String())
	t.currentDigest = wl.Digest()
	t.Plan.logger.Printf("%s Digest: %s\n", t.Name(), t.currentDigest)

	if t.alwaysBuild {
		return false
	}

	content, err := ioutil.ReadFile(t.SuccessMarkFile())
	if err != nil {
		t.Plan.logger.Printf("%s ExistDigest Error: %v", t.Name(), err)
		return false
	}
	prevDigest := strings.TrimSpace(string(content))
	t.Plan.logger.Printf("%s ExistDigest %s", t.Name(), prevDigest)
	if prevDigest == "" {
		return false
	}
	return t.currentDigest == prevDigest
}

// ClearSuccessMark removes the success mark
func (t *Task) ClearSuccessMark() error {
	t.alwaysBuild = true
	if t.Plan.DryRun {
		return nil
	}
	return os.Remove(t.SuccessMarkFile())
}

// BuildSuccessMark checks if the task can be skipped
func (t *Task) BuildSuccessMark() error {
	defer func() {
		t.currentDigest = ""
	}()
	if t.Result == Success && t.currentDigest != "" {
		return ioutil.WriteFile(t.SuccessMarkFile(), []byte(t.currentDigest), 0644)
	}
	return nil
}

// Runner gets runner according to exec-driver
func (t *Task) Runner() (Runner, error) {
	driver := t.Target.ExecDriver
	if driver == "" {
		if err := t.Target.GetSettings(SettingExecDriver, &driver); err != nil {
			return nil, err
		}
	}
	if driver == "" {
		driver = DefaultExecDriver
	}
	runner := runners[driver]
	if runner == nil {
		return nil, fmt.Errorf("invalid exec-driver: %s", driver)
	}
	return runner, nil
}

// SuccessMarkFile returns the filename of success mark
func (t *Task) SuccessMarkFile() string {
	return filepath.Join(t.Plan.WorkPath, t.Name()+".success")
}

// ScriptFile returns the filename of script
func (t *Task) ScriptFile() string {
	return filepath.Join(t.Plan.WorkPath, t.Name()+".script")
}

// LogFile returns the fullpath to log filename
func (t *Task) LogFile() string {
	return filepath.Join(t.Plan.WorkPath, t.Name()+".log")
}

// WorkingDir is absolute path of working dir to execute the task
func (t *Task) WorkingDir(dirs ...string) string {
	return filepath.Join(t.Project().BaseDir, t.Target.WorkingDir(dirs...))
}

// BuildScript generates script file according to cmds/script in target
func (t *Task) BuildScript() string {
	target := t.Target
	script := target.Script
	if script == "" && len(target.Cmds) > 0 {
		lines := make([]string, 0, len(target.Cmds))
		for _, cmd := range target.Cmds {
			if cmd == nil || cmd.Shell == "" {
				continue
			}
			lines = append(lines, cmd.Shell)
		}
		if len(lines) > 0 {
			script = "#!/bin/sh\nset -e\n" + strings.Join(lines, "\n") + "\n"
		}
	}
	return script
}

// WriteScriptFile builds the script file with provided script
func (t *Task) WriteScriptFile(script string) error {
	t.Plan.logger.Printf("%s WriteScript:\n%s\n", t.Name(), script)
	return ioutil.WriteFile(t.ScriptFile(), []byte(script), 0755)
}

// BuildScriptFile generates the script file using default generated script
func (t *Task) BuildScriptFile() (string, error) {
	script := t.BuildScript()
	return script, t.WriteScriptFile(script)
}

// Exec executes an external command for a task
func (t *Task) Exec(command string, args ...string) error {
	t.Plan.logger.Printf("%s Exec: %s %v\n", t.Name(), command, args)
	out, err := os.OpenFile(t.LogFile(),
		syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC,
		0644)
	if err != nil {
		t.Plan.logger.Printf("%s Exec OpenLog Error: %v\n", t.Name(), err)
		return err
	}
	defer out.Close()
	w := io.MultiWriter(out, t)
	cmd := exec.Command(command, args...)
	cmd.Env = append([]string{}, os.Environ()...)
	for name, value := range t.Plan.Env {
		cmd.Env = append(cmd.Env, name+"="+value)
	}
	cmd.Env = append(cmd.Env, "HMAKE_TARGET="+t.Name())
	cmd.Dir = filepath.Join(t.Project().BaseDir, t.Target.WorkingDir())
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

// ExecScript executes generated script
func (t *Task) ExecScript() error {
	return t.Exec(t.ScriptFile())
}

// Write implements io.Writer to receive execution output
func (t *Task) Write(p []byte) (int, error) {
	t.Plan.emit(&EvtTaskOutput{Task: t, Output: p})
	return len(p), nil
}
