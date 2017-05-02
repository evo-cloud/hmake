// +build windows

package docker

import (
	"fmt"
	"path/filepath"
	"strings"
)

func parseWindowsPath(path string) string {
	var slashedPath = filepath.ToSlash(path)
	var subPaths = strings.Split(slashedPath, ":")
	return fmt.Sprintf("/%v%v", strings.ToLower(subPaths[0]), subPaths[1])
}

func (r *Runner) canonicalProjectDir() string {
	return parseWindowsPath(r.projectDir)
}

func (r *Runner) exposeDocker() {
	r.exposeDockerEnv()
}
