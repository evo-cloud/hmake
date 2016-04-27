package project

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/easeway/langx.go/errors"
)

// ExecPlan describes the plan for execution
type ExecPlan struct {
	// Project is the wrapped project
	Project *Project
	// Tasks are the tasks need execution
	Tasks map[string]*Task
	// MaxConcurrency defines maximum number of tasks being executed in parallel
	// if it's 0, the number of CPU cores are counted
	MaxConcurrency int
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

	finishCh chan *Task
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
}

// TaskResult indicates the result of task execution
type TaskResult int

// Task results
const (
	Success TaskResult = iota
	Failure
	Skipped
)

func (r TaskResult) String() string {
	switch r {
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

// Runner is the handler execute a target
type Runner func(*Task) (TaskResult, error)

// Setting Names
const (
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
	return &ExecPlan{
		Project:      project,
		Tasks:        make(map[string]*Task),
		WaitingTasks: make(map[string]*Task),
	}
}

// OnEvent subscribes the events
func (p *ExecPlan) OnEvent(handler EventHandler) *ExecPlan {
	p.EventHandler = handler
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
		} else {
			p.AddTarget(t)
		}
	}
	return errs.Aggregate()
}

// AddTarget adds a target into execution plan
func (p *ExecPlan) AddTarget(t *Target) *Task {
	task, exists := p.Tasks[t.Name]
	if !exists {
		task = NewTask(p, t)
		p.Tasks[t.Name] = task
		for name, dep := range t.Depends {
			task.Depends[name] = p.AddTarget(dep)
		}
		if task.IsActivated() {
			task.State = Queued
			p.QueuedTasks = append(p.QueuedTasks, task)
		} else {
			task.State = Waiting
			p.WaitingTasks[t.Name] = task
		}
	}
	return task
}

// Execute start execution
func (p *ExecPlan) Execute() error {
	concurrency := p.MaxConcurrency
	if concurrency == 0 {
		concurrency = runtime.NumCPU()
	}

	if err := p.Project.ExecPrepare(); err != nil {
		return err
	}

	errs := &errors.AggregatedError{}
	p.finishCh = make(chan *Task)
	p.RunningTasks = make(map[string]*Task)

	for _, task := range p.QueuedTasks {
		p.emit(&EvtTaskActivated{Task: task})
	}

	for {
		tasks := p.dequeueTasks(concurrency)
		if len(tasks) > 0 {
			for _, task := range tasks {
				p.startTask(task, errs)
			}
		} else if len(p.RunningTasks) == 0 {
			// nothing to run
			break
		}

		p.finishTask(<-p.finishCh, errs)
	}
	return errs.Aggregate()
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
	task.State = Running
	p.RunningTasks[task.Name()] = task
	p.emit(&EvtTaskStart{Task: task})
	runner, err := task.Runner()
	if err != nil {
		task.Error = err
		task.Result = Failure
		p.finishTask(task, errs)
	} else {
		go p.run(task, runner)
	}
}

func (p *ExecPlan) run(task *Task, runner Runner) {
	task.Result, task.Error = runner(task)
	p.finishCh <- task
}

func (p *ExecPlan) finishTask(task *Task, errs *errors.AggregatedError) {
	if _, exist := p.RunningTasks[task.Name()]; !exist {
		// task is out-of-date, ignored
		return
	}

	// transit to finished state
	task.State = Finished
	delete(p.RunningTasks, task.Name())
	p.FinishedTasks = append(p.FinishedTasks, task)

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
	return t.Plan.Project
}

// IsActivated indicates the task is ready to run
func (t *Task) IsActivated() bool {
	return len(t.Depends) == 0
}

// Runner gets runner according to exec-driver
func (t *Task) Runner() (Runner, error) {
	driver := t.Target.ExecDriver
	if driver == "" {
		if err := t.Target.GetSetting(SettingExecDriver, &driver); err != nil {
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

// ScriptFile returns the filename of script
func (t *Task) ScriptFile() string {
	return filepath.Join(t.Project().WorkPath(), t.Name()+".script")
}

// LogFile returns the fullpath to log filename
func (t *Task) LogFile() string {
	return filepath.Join(t.Project().WorkPath(), t.Name()+".log")
}

// GenerateScript generates script file according to cmds/script in target
func (t *Task) GenerateScript() (string, error) {
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
			script = "#!/bin/sh\n" + strings.Join(lines, "\n") + "\n"
		}
	}
	if script == "" {
		return "", nil
	}

	return script, ioutil.WriteFile(t.ScriptFile(), []byte(script), 0755)
}

// Exec executes an external command for a task
func (t *Task) Exec(command string, args ...string) error {
	out, err := os.OpenFile(t.LogFile(),
		syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC,
		0644)
	if err != nil {
		return err
	}
	defer out.Close()
	w := io.MultiWriter(out, t)
	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()
	cmd.Dir = t.Project().BaseDir
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
