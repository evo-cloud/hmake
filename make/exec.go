package make

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/easeway/clix.go"
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

	finishCh chan *Task
}

// Task is the execution state of a target
type Task struct {
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
type Runner func(*Target) (TaskResult, error)

// NewExecPlan creates an ExecPlan for a Project
func NewExecPlan(project *Project) *ExecPlan {
	return &ExecPlan{
		Project:      project,
		Tasks:        make(map[string]*Task),
		WaitingTasks: make(map[string]*Task),
	}
}

// Require adds targets to be executed
func (p *ExecPlan) Require(targets ...string) error {
	if p.Tasks == nil {
		p.Tasks = make(map[string]*Task)
	}
	if p.WaitingTasks == nil {
		p.WaitingTasks = make(map[string]*Task)
	}
	errs := &clix.AggregatedError{}
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
		task = NewTask(t)
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
func (p *ExecPlan) Execute(runner Runner) error {
	concurrency := p.MaxConcurrency
	if concurrency == 0 {
		concurrency = runtime.NumCPU()
	}

	errs := &clix.AggregatedError{}
	p.finishCh = make(chan *Task)
	p.RunningTasks = make(map[string]*Task)
	for {
		maxDequeueCount := concurrency
		if maxDequeueCount < 0 {
			maxDequeueCount = len(p.QueuedTasks)
		} else {
			maxDequeueCount -= len(p.RunningTasks)
			if l := len(p.QueuedTasks); maxDequeueCount > l {
				maxDequeueCount = l
			}
		}
		if maxDequeueCount > 0 {
			tasks := p.QueuedTasks[0:maxDequeueCount]
			p.QueuedTasks = p.QueuedTasks[maxDequeueCount:]
			for _, task := range tasks {
				task.State = Running
				p.RunningTasks[task.Name()] = task
				go p.run(task, runner)
			}
		} else if len(p.RunningTasks) == 0 {
			if len(p.WaitingTasks) > 0 {
				names := make([]string, 0, len(p.WaitingTasks))
				for name := range p.WaitingTasks {
					names = append(names, name)
				}
				errs.Add(fmt.Errorf("some tasks can't be activated: %s",
					strings.Join(names, ",")))
			}
			break
		}

		task := <-p.finishCh
		if _, exist := p.RunningTasks[task.Name()]; exist {
			task.State = Finished
			delete(p.RunningTasks, task.Name())
			p.FinishedTasks = append(p.FinishedTasks, task)
			for name := range task.Target.Activates {
				if t := p.Tasks[name]; t != nil {
					delete(t.Depends, task.Name())
					if t.IsActivated() && p.WaitingTasks[t.Name()] != nil {
						delete(p.WaitingTasks, t.Name())
						t.State = Queued
						p.QueuedTasks = append(p.QueuedTasks, t)
					}
				}
			}
			errs.Add(task.Error)
		}
	}
	return errs.Aggregate()
}

func (p *ExecPlan) run(task *Task, runner Runner) {
	task.Result, task.Error = runner(task.Target)
	p.finishCh <- task
}

// NewTask creates a task for a target
func NewTask(t *Target) *Task {
	return &Task{Target: t, Depends: make(map[string]*Task)}
}

// Name returns the name of wrapped target
func (t *Task) Name() string {
	return t.Target.Name
}

// IsActivated indicates the task is ready to run
func (t *Task) IsActivated() bool {
	return len(t.Depends) == 0
}
