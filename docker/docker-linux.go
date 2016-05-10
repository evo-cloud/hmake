// +build linux

package docker

import "os"

const (
	dockerSockPath = "/var/run/docker.sock"
)

func (c *dockerConfig) exposeDocker(r *dockerRunner) {
	c.exposeDockerEnv()
	if os.Getenv("DOCKER_HOST") == "" {
		sockPath := dockerSockPath
		if r.task.Target.Ext != nil {
			serverSock, ok := r.task.Target.Ext["server-socket"].(string)
			if ok && serverSock != "" {
				sockPath = serverSock
			}
		}
		c.Volumes = append(c.Volumes, sockPath+":"+sockPath)
	}
}

func currentUserIds() (uid, gid int, grps []int, err error) {
	uid = os.Getuid()
	gid = os.Getgid()
	grps, err = os.Getgroups()
	return
}
