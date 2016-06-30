// +build darwin

package docker

func (r *Runner) checkProjectDir() error {
	return nil
}

func (r *Runner) canonicalProjectDir() string {
	return r.projectDir
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
