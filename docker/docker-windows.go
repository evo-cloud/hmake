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

func (r *Runner) checkProjectDir() error {
	if !strings.HasPrefix(r.projectDir, "C:\\Users\\") {
		return fmt.Errorf("The project path must be prefixed with C:\\Users\\ on Windows")
	}

	return nil
}

func (r *Runner) canonicalProjectDir() string {
	return parseWindowsPath(r.projectDir)
}

func (r *Runner) exposeDocker() {
	r.exposeDockerEnv()
}

func currentUserIds() (uid, gid int, grps []int, err error) {
	return currentUserIdsFromDockerMachine()
}

func userID(name string) (uid, gid int, err error) {
	return userIDFromDockerMachine(name)
}
