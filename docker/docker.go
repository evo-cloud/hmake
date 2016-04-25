package docker

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/evo-cloud/hmake/make"
)

const (
	// DefaultSrcVolume is the default source path inside the container
	DefaultSrcVolume = "/root/src"
)

type dockerConfig struct {
	Image     string   `json:"image"`
	SrcVolume string   `json:"src-volume"`
	Envs      []string `json:"envs"`
}

// NewRunner creates docker backed runner
func NewRunner(project *make.Project) make.Runner {
	return func(t *make.Target) (make.TaskResult, error) {
		err := dockerRun(project, t)
		if err != nil {
			return make.Failure, err
		}
		return make.Success, nil
	}
}

func dockerRun(project *make.Project, target *make.Target) error {
	if target.Script == "" {
		return nil
	}

	scriptFile := target.Name + ".script"
	err := ioutil.WriteFile(
		filepath.Join(project.WorkPath(), scriptFile),
		[]byte(target.Script), 0755)
	if err != nil {
		return err
	}
	dockerRun := []string{"docker", "run",
		"-a", "STDOUT", "-a", "STDERR",
		"--rm",
	}

	var conf dockerConfig
	if err = project.GetSettings(&conf); err != nil {
		return err
	}
	if err = target.GetExt(&conf); err != nil {
		return err
	}
	if conf.Image == "" {
		return target.Errorf("missing property image")
	}
	if conf.SrcVolume == "" {
		conf.SrcVolume = DefaultSrcVolume
	}
	dockerRun = append(dockerRun,
		"-v", project.BaseDir+":"+conf.SrcVolume,
		"-w", conf.SrcVolume,
		"--entrypoint", filepath.Join(conf.SrcVolume, project.WorkFolder, scriptFile))
	for _, env := range target.Envs {
		pos := strings.Index(env, "=")
		var name string
		if pos > 0 {
			name = env[0:pos]
		} else {
			name = env
		}
		found := false
		for n, confEnv := range conf.Envs {
			if name == confEnv || strings.HasPrefix(confEnv, name+"=") {
				rest := conf.Envs[n+1:]
				conf.Envs = append(append(conf.Envs[0:n], env), rest...)
				found = true
				break
			}
		}
		if !found {
			conf.Envs = append(conf.Envs, env)
		}
	}
	for _, env := range conf.Envs {
		dockerRun = append(dockerRun, "-e", env)
	}

	dockerRun = append(dockerRun, conf.Image)

	if dockerRun[0], err = exec.LookPath(dockerRun[0]); err != nil {
		return err
	}

	fmt.Println(strings.Join(dockerRun, " "))
	cmd := exec.Cmd{
		Path:   dockerRun[0],
		Args:   dockerRun,
		Env:    os.Environ(),
		Dir:    project.BaseDir,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	return cmd.Run()
}
