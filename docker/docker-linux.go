// +build linux

package docker

import (
	"os"
	"os/user"
	"strconv"
)

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

func currentUserIds() (uid, gid int, grps []int, err error) {
	uid = os.Getuid()
	gid = os.Getgid()
	grps, err = os.Getgroups()
	return
}

func userID(name string) (uid, gid int, err error) {
	var u *user.User
	if uid, err = strconv.Atoi(name); err == nil {
		u, err = user.LookupId(name)
	} else {
		u, err = user.Lookup(name)
	}
	if err == nil {
		uid, err = strconv.Atoi(u.Uid)
		if err == nil {
			gid, err = strconv.Atoi(u.Gid)
		}
	}
	return
}
