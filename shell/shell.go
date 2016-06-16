package shell

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	hm "github.com/evo-cloud/hmake/project"
)

// Target defines the schema of shell commands in target
type Target struct {
	Console bool       `json:"console"`
	Env     []string   `json:"env"`
	Cmds    []*Command `json:"cmds"`
	Script  string     `json:"script"`
}

// Command defines a single command to execute
type Command struct {
	Shell string                 `json:"*"`
	Ext   map[string]interface{} `json:"*"`
}

// ScriptFile returns the filename of script
func ScriptFile(t *hm.Task) string {
	return filepath.Join(t.Plan.WorkPath, t.Name()+".script")
}

// LogFile returns the fullpath to log filename
func LogFile(t *hm.Task) string {
	return filepath.Join(t.Plan.WorkPath, t.Name()+".log")
}

// BuildScript generates script file according to cmds/script in target
func BuildScript(t *hm.Task) string {
	var target Target
	t.Target.GetExt(&target)
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
func WriteScriptFile(t *hm.Task, script string) error {
	t.Plan.Logf("%s WriteScript:\n%s", t.Name(), script)
	return ioutil.WriteFile(ScriptFile(t), []byte(script), 0755)
}

// BuildScriptFile generates the script file using default generated script
func BuildScriptFile(t *hm.Task) (string, error) {
	script := BuildScript(t)
	return script, WriteScriptFile(t, script)
}

// Executor wraps over exec.Cmd with output file
type Executor struct {
	Task    *hm.Task
	Cmd     *exec.Cmd
	Console bool
	Output  bool
}

// Mute disables the output
func (x *Executor) Mute() *Executor {
	x.Output = false
	return x
}

// Run starts the executor
func (x *Executor) Run(sigCh <-chan os.Signal) (err error) {
	x.Task.Plan.Logf("%s Exec: %v\n", x.Task.Name(), x.Cmd.Args)

	if x.Console {
		x.Cmd.Stdin = os.Stdin
		x.Cmd.Stdout = os.Stdout
		x.Cmd.Stderr = os.Stderr
	} else if x.Output {
		var out *os.File
		out, err = os.OpenFile(LogFile(x.Task),
			syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC,
			0644)
		if err != nil {
			x.Task.Plan.Logf("%s Exec OpenLog Error: %v\n", x.Task.Name(), err)
			return err
		}
		defer out.Close()
		w := io.MultiWriter(out, x.Task)
		x.Cmd.Stdout = w
		x.Cmd.Stderr = w
	}

	if sigCh == nil {
		return x.Cmd.Run()
	}

	if err = x.Cmd.Start(); err != nil {
		return
	}

	runCh := make(chan error)
	go func() {
		runCh <- x.Cmd.Wait()
	}()

	for {
		select {
		case err = <-runCh:
			return
		case sig := <-sigCh:
			x.Cmd.Process.Signal(sig)
		}
	}
}

// Exec executes an external command for a task
func Exec(t *hm.Task, command string, args ...string) *Executor {
	var target Target
	t.Target.GetExt(&target)
	cmd := exec.Command(command, args...)
	cmd.Env = append([]string{}, os.Environ()...)
	cmd.Env = append(cmd.Env, target.Env...)
	for name, value := range t.Plan.Env {
		cmd.Env = append(cmd.Env, name+"="+value)
	}
	cmd.Env = append(cmd.Env, t.EnvVars()...)
	cmd.Dir = filepath.Join(t.Project().BaseDir, t.Target.WorkingDir())
	return &Executor{Task: t, Cmd: cmd, Console: target.Console, Output: true}
}

// ExecScript executes generated script
func ExecScript(t *hm.Task) *Executor {
	return Exec(t, ScriptFile(t))
}

const (
	// ExecDriverName is the name of exec-driver
	ExecDriverName = "shell"
)

// Runner is shell runner
type Runner struct {
	Task *hm.Task
}

// Run implements Runner
func (r *Runner) Run(sigCh <-chan os.Signal) (hm.TaskResult, error) {
	script, err := BuildScriptFile(r.Task)
	if err != nil {
		return hm.Failure, err
	}
	if script == "" {
		return hm.Success, nil
	}
	err = ExecScript(r.Task).Run(sigCh)
	if err != nil {
		return hm.Failure, err
	}
	return hm.Success, nil
}

// Factory is runner factory
func Factory(task *hm.Task) (hm.Runner, error) {
	return &Runner{Task: task}, nil
}

func init() {
	hm.RegisterExecDriver(ExecDriverName, Factory)
}
