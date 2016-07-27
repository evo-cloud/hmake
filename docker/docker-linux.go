// +build linux

package docker

import "os"

const (
	dockerSockPath = "/var/run/docker.sock"
)

func (r *Runner) checkProjectDir() error {
	return nil
}

func (r *Runner) canonicalProjectDir() string {
	return r.projectDir
}

func (r *Runner) exposeDocker() {
	r.exposeDockerEnv()
	if os.Getenv("DOCKER_HOST") == "" {
		sockPath := dockerSockPath
		if r.Task.Target.Ext != nil {
			serverSock, ok := r.Task.Target.Ext["server-socket"].(string)
			if ok && serverSock != "" {
				sockPath = serverSock
			}
		}
		r.Volumes = append(r.Volumes, sockPath+":"+sockPath)
	}
}
