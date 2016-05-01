package shell

import hm "github.com/evo-cloud/hmake/project"

const (
	// ExecDriverName is the name of exec-driver
	ExecDriverName = "shell"
)

// Runner wraps docker implementation
func Runner(task *hm.Task) (hm.TaskResult, error) {
	script, err := task.BuildScriptFile()
	if err == nil && script != "" {
		err = task.ExecScript()
	}
	if err != nil {
		return hm.Failure, err
	}
	return hm.Success, nil
}

func init() {
	hm.RegisterExecDriver(ExecDriverName, Runner)
}
