// +build darwin

package docker

import (
	"fmt"
	"strings"
)

func (r *Runner) checkProjectDir() error {
	if !strings.HasPrefix(r.projectDir, "/Users/") {
		return fmt.Errorf("The project path must be prefixed with /Users/ on Mac OS")
	}

	return nil
}

func (r *Runner) canonicalProjectDir() string {
	return r.projectDir
}

func (r *Runner) exposeDocker() {
	r.exposeDockerEnv()
}
