package docker

import (
	"path/filepath"
	"strings"

	hm "github.com/evo-cloud/hmake/project"
	"github.com/evo-cloud/hmake/shell"
)

const (
	// ExecDriverName is name of exec-driver
	ExecDriverName = "docker"
	// DefaultSrcVolume is the default source path inside the container
	DefaultSrcVolume = "/root/src"
)

type dockerConfig struct {
	Image     string   `json:"image"`
	SrcVolume string   `json:"src-volume"`
	Envs      []string `json:"envs"`
}

func (c *dockerConfig) addEnv(envs ...string) {
	for _, env := range envs {
		pos := strings.Index(env, "=")
		var name string
		if pos > 0 {
			name = env[0:pos]
		} else {
			name = env
		}
		found := false
		for n, confEnv := range c.Envs {
			if name == confEnv || strings.HasPrefix(confEnv, name+"=") {
				c.Envs[n] = env
				found = true
				break
			}
		}
		if !found {
			c.Envs = append(c.Envs, env)
		}
	}
}

type dockerRunner struct {
	task *hm.Task
}

// Runner wraps docker implementation
func Runner(task *hm.Task) (hm.TaskResult, error) {
	r := &dockerRunner{task: task}
	conf, err := r.loadConfig()
	if err == nil && conf.Image == "" {
		// image not present, fallback to shell
		return shell.Runner(task)
	}
	if err = r.run(conf); err != nil {
		return hm.Failure, err
	}
	return hm.Success, nil
}

func (r *dockerRunner) loadConfig() (conf *dockerConfig, err error) {
	conf = &dockerConfig{}

	project := r.task.Project()
	target := r.task.Target

	if err = project.GetSettings(conf); err != nil {
		return
	}
	if err = target.GetExt(conf); err != nil {
		return
	}
	if conf.SrcVolume == "" {
		conf.SrcVolume = DefaultSrcVolume
	}
	conf.addEnv(target.Envs...)
	for name, value := range r.task.Plan.Env {
		conf.addEnv(name + "=" + value)
	}
	conf.addEnv("HMAKE_PROJECT_DIR=" + conf.SrcVolume)
	conf.addEnv("HMAKE_PROJECT_FILE=" +
		filepath.Join(conf.SrcVolume,
			filepath.Base(r.task.Project().MasterFile.Source)))
	conf.addEnv("HMAKE_WORK_DIR=" +
		filepath.Join(conf.SrcVolume,
			filepath.Base(r.task.Plan.WorkPath)))
	conf.addEnv("HMAKE_TARGET=" + r.task.Name())
	return
}

func (r *dockerRunner) run(conf *dockerConfig) error {
	project := r.task.Project()

	dockerRun := []string{"docker", "run",
		"-a", "STDOUT", "-a", "STDERR",
		"--rm",
		"-v", project.BaseDir + ":" + conf.SrcVolume,
		"-w", conf.SrcVolume,
		"--entrypoint", filepath.Join(conf.SrcVolume, hm.WorkFolder, filepath.Base(r.task.ScriptFile())),
	}
	for _, env := range conf.Envs {
		dockerRun = append(dockerRun, "-e", env)
	}
	dockerRun = append(dockerRun, conf.Image)

	script, err := r.task.BuildScriptFile()
	if err != nil || script == "" {
		return err
	}

	return r.task.Exec(dockerRun[0], dockerRun[1:]...)
}

func init() {
	hm.RegisterExecDriver(ExecDriverName, Runner)
}
