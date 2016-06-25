package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var (
	errDockerMachineUnknown = fmt.Errorf("unknown DOCKER_MACHINE_NAME")
)

func inspectIds(opt string) (ids []int, err error) {
	machine := os.Getenv("DOCKER_MACHINE_NAME")
	if machine == "" {
		return ids, errDockerMachineUnknown
	}

	args := []string{"docker-machine", "ssh", machine, "id", opt}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		return ids, fmt.Errorf("error: [%s]: %v", strings.Join(args, " "), err)
	}
	tokens := strings.Split(string(out), " ")
	for _, token := range tokens {
		if i, err := strconv.Atoi(strings.TrimSpace(token)); err == nil {
			ids = append(ids, i)
		}
	}
	if len(ids) == 0 {
		return ids, fmt.Errorf("no id found: [%s]", strings.Join(args, " "))
	}
	return
}

func currentUserIdsFromDockerMachine() (uid, gid int, grps []int, err error) {
	var uids, gids []int
	if uids, err = inspectIds("-u"); err != nil {
		return
	}
	uid = uids[0]
	if gids, err = inspectIds("-g"); err != nil {
		return
	}
	gid = gids[0]
	if grps, err = inspectIds("-G"); err != nil {
		return
	}
	return
}
