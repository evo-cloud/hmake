// +build darwin

package docker

func (c *dockerConfig) exposeDocker(*dockerRunner) {
	c.exposeDockerEnv()
}
