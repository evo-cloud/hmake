// +build darwin

package docker

func (r *Runner) canonicalProjectDir() string {
	return r.projectDir
}

func (r *Runner) exposeDocker() {
	r.exposeDockerEnv()
}
