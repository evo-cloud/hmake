package docker

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	hm "github.com/evo-cloud/hmake/project"
	"github.com/evo-cloud/hmake/shell"
)

const (
	// ExecDriverName is name of exec-driver
	ExecDriverName = "docker"
	// DefaultSrcVolume is the default source path inside the container
	DefaultSrcVolume = "/src"
	// SettingName is the name of section of settings
	SettingName = "docker"
	// Dockerfile is default name of Dockerfile
	Dockerfile = "Dockerfile"
)

// Runner is a docker runner
type Runner struct {
	Task *hm.Task `json:"-"`

	Build             string   `json:"build"`
	BuildFrom         string   `json:"build-from"`
	Image             string   `json:"image"`
	SrcVolume         string   `json:"src-volume"`
	ExposeDocker      bool     `json:"expose-docker"`
	Envs              []string `json:"envs"`
	EnvFiles          []string `json:"env-files"`
	CapAdd            []string `json:"cap-add"`
	CapDrop           []string `json:"cap-drop"`
	Devices           []string `json:"devices"`
	Privileged        bool     `json:"privileged"`
	Network           string   `json:"net"`
	Hosts             []string `json:"hosts"`
	DNSServers        []string `json:"dns"`
	DNSSearch         string   `json:"dns-search"`
	DNSOpts           []string `json:"dns-opts"`
	User              string   `json:"user"`
	Groups            []string `json:"groups"`
	Volumes           []string `json:"volumes"`
	BlkIoWeight       *int     `json:"blkio-weight"`
	BlkIoWeightDevs   []string `json:"blkio-weight-devices"`
	DevReadBps        []string `json:"device-read-bps"`
	DevWriteBps       []string `json:"device-write-bps"`
	DevReadIops       []string `json:"device-read-iops"`
	DevWriteIops      []string `json:"device-write-iops"`
	CPUShares         *int     `json:"cpu-shares"`
	CPUPeriod         *int     `json:"cpu-period"`
	CPUQuota          *int     `json:"cpu-quota"`
	CPUSetCPUs        string   `json:"cpuset-cpus"`
	CPUSetMems        string   `json:"cpuset-mems"`
	KernelMemory      string   `json:"kernel-memory"`
	Memory            string   `json:"memory"`
	MemorySwap        string   `json:"memory-swap"`
	MemoryReservation string   `json:"memory-reservation"`
	MemorySwappiness  *int     `json:"memory-swappiness"`
	ShmSize           string   `json:"shm-size"`

	projectDir string
}

func (r *Runner) addEnv(envs ...string) {
	for _, env := range envs {
		if env == "" {
			continue
		}
		pos := strings.Index(env, "=")
		var name string
		if pos > 0 {
			name = env[0:pos]
		} else {
			name = env
		}
		found := false
		for n, confEnv := range r.Envs {
			if name == confEnv || strings.HasPrefix(confEnv, name+"=") {
				r.Envs[n] = env
				found = true
				break
			}
		}
		if !found {
			r.Envs = append(r.Envs, env)
		}
	}
}

func (r *Runner) exposeDockerEnv() {
	if val := os.Getenv("DOCKER_HOST"); val != "" {
		r.addEnv("DOCKER_HOST=" + val)
	}
	if certPath := os.Getenv("DOCKER_CERT_PATH"); certPath != "" {
		r.addEnv("DOCKER_CERT_PATH=" + certPath)
		r.Volumes = append(r.Volumes, certPath+":"+certPath)
	}
	if val := os.Getenv("DOCKER_TLS_VERIFY"); val != "" {
		r.addEnv("DOCKER_TLS_VERIFY=" + val)
	}
}

func (r *Runner) cidFile() string {
	return filepath.Join(r.Task.Plan.WorkPath, r.Task.Name()+".cid")
}

func (r *Runner) cid() (cid string) {
	data, err := ioutil.ReadFile(r.cidFile())
	if err == nil && data != nil {
		cid = string(data)
	}
	return
}

func (r *Runner) signal(sig os.Signal) {
	sysSig := sig.(syscall.Signal)
	if cid := r.cid(); cid != "" {
		r.Task.Plan.Logf("Signal container %s %d", cid, sysSig)
		// HACK: in non-tty mode, docker is not going to pass the signal to init
		// process, then the INTERRUPT/TERM signal should be translated into kill
		if sysSig == syscall.SIGINT || sysSig == syscall.SIGTERM {
			shell.Exec(r.Task, "docker", "kill", cid).Mute().Run(nil)
		} else {
			shell.Exec(r.Task, "docker", "kill", "-s", strconv.Itoa(int(sysSig)), cid).
				Mute().
				Run(nil)
		}
	} else {
		r.Task.Plan.Logf("Ignore signal %d, CID not available", sysSig)
	}
}

func (r *Runner) removeContainer() {
	if cid := r.cid(); cid != "" {
		r.Task.Plan.Logf("Removing container %s", cid)
		shell.Exec(r.Task, "docker", "rm", "-f", cid).Mute().Run(nil)
	} else {
		r.Task.Plan.Logf("Ignore removing container, CID not available")
	}
}

// Run implements Runner
func (r *Runner) Run(sigCh <-chan os.Signal) (result hm.TaskResult, err error) {
	result = hm.Success

	os.Remove(r.cidFile())
	if r.Build != "" {
		err = r.build(sigCh)
	}
	if err == nil {
		err = r.run(sigCh)
	}
	if err != nil {
		result = hm.Failure
	}
	r.removeContainer()
	return
}

func (r *Runner) build(sigCh <-chan os.Signal) error {
	dockerCmd := []string{"docker", "build", "-t", r.Image}

	dockerFile := r.Task.WorkingDir(r.Build)
	buildFrom := r.BuildFrom
	if buildFrom != "" {
		buildFrom = r.Task.WorkingDir(buildFrom)
	}

	info, err := os.Stat(dockerFile)
	if err != nil {
		return err
	}

	if info.IsDir() {
		if buildFrom == "" {
			dockerCmd = append(dockerCmd, dockerFile)
		} else {
			dockerCmd = append(dockerCmd,
				"-f", filepath.Join(dockerFile, Dockerfile),
				buildFrom)
		}
	} else if buildFrom == "" {
		dockerCmd = append(dockerCmd,
			"-f", dockerFile,
			filepath.Dir(dockerFile))
	} else {
		dockerCmd = append(dockerCmd,
			"-f", dockerFile,
			buildFrom)
	}

	return shell.Exec(r.Task, dockerCmd[0], dockerCmd[1:]...).Run(sigCh)
}

func (r *Runner) run(sigCh <-chan os.Signal) error {
	workDir := filepath.Join(r.SrcVolume, r.Task.Target.WorkingDir())
	dockerRun := []string{"docker", "run",
		"-a", "STDOUT", "-a", "STDERR",
		"--rm",
		"-v", r.projectDir + ":" + r.SrcVolume,
		"-w", workDir,
		"--entrypoint", filepath.Join(r.SrcVolume, hm.WorkFolder,
			filepath.Base(shell.ScriptFile(r.Task))),
		"--cidfile", r.cidFile(),
	}
	// by default, use non-root user
	if r.User == "" {
		uid, gid, grps, err := currentUserIds()
		if err != nil {
			return err
		}
		dockerRun = append(dockerRun, "-u", strconv.Itoa(uid)+":"+strconv.Itoa(gid))
		if len(r.Groups) == 0 {
			for _, grp := range grps {
				if grp != gid {
					dockerRun = append(dockerRun, "--group-add", strconv.Itoa(grp))
				}
			}
		}
	} else if r.User != "root" && r.User != "0" {
		dockerRun = append(dockerRun, "-u", r.User)
	}
	for _, grp := range r.Groups {
		dockerRun = append(dockerRun, "--group-add", grp)
	}

	for _, envFile := range r.EnvFiles {
		dockerRun = append(dockerRun, "--env-file",
			filepath.Join(r.SrcVolume, r.Task.Target.WorkingDir(envFile)))
	}

	for _, env := range r.Envs {
		dockerRun = append(dockerRun, "-e", env)
	}

	if r.Network == "host" {
		dockerRun = append(dockerRun, "--net=host", "--uts=host")
	} else {
		for _, host := range r.Hosts {
			dockerRun = append(dockerRun, "--add-host", host)
		}
		for _, dns := range r.DNSServers {
			dockerRun = append(dockerRun, "--dns", dns)
		}
		if r.DNSSearch != "" {
			dockerRun = append(dockerRun, "--dns-search", r.DNSSearch)
		}
		for _, opt := range r.DNSOpts {
			dockerRun = append(dockerRun, "--dns-opt", opt)
		}
	}

	for _, cap := range r.CapAdd {
		dockerRun = append(dockerRun, "--cap-add", cap)
	}
	for _, cap := range r.CapDrop {
		dockerRun = append(dockerRun, "--cap-drop", cap)
	}

	for _, dev := range r.Devices {
		dockerRun = append(dockerRun, "--device", dev)
	}

	if r.Privileged {
		dockerRun = append(dockerRun, "--privileged")
	}

	for _, vol := range r.Volumes {
		hostVol := vol
		if !filepath.IsAbs(hostVol) {
			hostVol = filepath.Join(r.projectDir, r.Task.Target.WorkingDir(vol))
		}
		dockerRun = append(dockerRun, "-v", hostVol)
	}

	if r.BlkIoWeight != nil {
		dockerRun = append(dockerRun, "--blkio-weight", strconv.Itoa(*r.BlkIoWeight))
	}
	for _, w := range r.BlkIoWeightDevs {
		dockerRun = append(dockerRun, "--blkio-weight-device", w)
	}
	for _, bps := range r.DevReadBps {
		dockerRun = append(dockerRun, "--device-read-bps", bps)
	}
	for _, bps := range r.DevWriteBps {
		dockerRun = append(dockerRun, "--device-write-bps", bps)
	}
	for _, iops := range r.DevReadIops {
		dockerRun = append(dockerRun, "--device-read-iops", iops)
	}
	for _, iops := range r.DevWriteIops {
		dockerRun = append(dockerRun, "--device-write-iops", iops)
	}
	if r.CPUShares != nil {
		dockerRun = append(dockerRun, "-c", strconv.Itoa(*r.CPUShares))
	}
	if r.CPUPeriod != nil {
		dockerRun = append(dockerRun, "--cpu-period", strconv.Itoa(*r.CPUPeriod))
	}
	if r.CPUQuota != nil {
		dockerRun = append(dockerRun, "--cpu-quota", strconv.Itoa(*r.CPUQuota))
	}
	if r.CPUSetCPUs != "" {
		dockerRun = append(dockerRun, "--cpuset-cpus", r.CPUSetCPUs)
	}
	if r.CPUSetMems != "" {
		dockerRun = append(dockerRun, "--cpuset-mems", r.CPUSetMems)
	}
	if r.KernelMemory != "" {
		dockerRun = append(dockerRun, "--kernel-memory", r.KernelMemory)
	}
	if r.Memory != "" {
		dockerRun = append(dockerRun, "-m", r.Memory)
	}
	if r.MemorySwap != "" {
		dockerRun = append(dockerRun, "--memory-swap", r.MemorySwap)
	}
	if r.MemorySwappiness != nil {
		dockerRun = append(dockerRun, "--memory-swappiness", strconv.Itoa(*r.MemorySwappiness))
	}
	if r.MemoryReservation != "" {
		dockerRun = append(dockerRun, "--memory-reservation", r.MemoryReservation)
	}
	if r.ShmSize != "" {
		dockerRun = append(dockerRun, "--shm-size", r.ShmSize)
	}

	dockerRun = append(dockerRun, r.Image)

	script, err := shell.BuildScriptFile(r.Task)
	if err != nil || script == "" {
		return err
	}

	x := shell.Exec(r.Task, dockerRun[0], dockerRun[1:]...)
	x.Cmd.Stdin = os.Stdin

	ch := make(chan struct{})
	go func() {
		select {
		case <-ch:
			return
		case sig := <-sigCh:
			r.signal(sig)
		}
	}()
	err = x.Run(nil)
	close(ch)
	return err
}

// Factory is runner factory
func Factory(task *hm.Task) (hm.Runner, error) {
	r := &Runner{Task: task}

	if err := task.Target.GetSettingsWithExt(SettingName, r); err != nil {
		return nil, err
	}

	if r.Image == "" {
		// image not present, fallback to shell
		return shell.Factory(task)
	}

	if r.SrcVolume == "" {
		r.SrcVolume = DefaultSrcVolume
	}
	if r.ExposeDocker {
		r.exposeDocker()
	}
	r.addEnv(task.Target.Envs...)
	for name, value := range task.Plan.Env {
		r.addEnv(name + "=" + value)
	}
	r.addEnv("HMAKE_PROJECT_DIR=" + r.SrcVolume)
	r.addEnv("HMAKE_PROJECT_FILE=" +
		filepath.Join(r.SrcVolume,
			filepath.Base(task.Project().MasterFile.Source)))
	r.addEnv("HMAKE_WORK_DIR=" +
		filepath.Join(r.SrcVolume,
			filepath.Base(task.Plan.WorkPath)))
	r.addEnv(task.Envs()...)

	r.projectDir = filepath.Clean(task.Project().BaseDir)
	volHost := os.Getenv("HMAKE_DOCKER_VOL_HOST")
	volCntr := os.Getenv("HMAKE_DOCKER_VOL_CNTR")
	// in nested situation
	if volHost != "" && volCntr != "" {
		prefix := filepath.Clean(volCntr) + "/"
		if strings.HasPrefix(r.projectDir, prefix) {
			r.projectDir = filepath.Join(volHost, r.projectDir[len(prefix):])
		}
	}
	r.addEnv("HMAKE_DOCKER_VOL_HOST=" + r.projectDir)
	r.addEnv("HMAKE_DOCKER_VOL_CNTR=" + r.SrcVolume)
	return r, nil
}

func init() {
	hm.RegisterExecDriver(ExecDriverName, Factory)
}
