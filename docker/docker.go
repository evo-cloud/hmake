package docker

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	hm "github.com/evo-cloud/hmake/project"
	"github.com/evo-cloud/hmake/shell"
)

const (
	// ExecDriverName is name of exec-driver
	ExecDriverName = "docker"
	// DefaultSrcVolume is the default source path inside the container
	DefaultSrcVolume = "/root/src"
	// SettingName is the name of section of settings
	SettingName = "docker"
)

type dockerConfig struct {
	Image             string   `json:"image"`
	SrcVolume         string   `json:"src-volume"`
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
}

func (c *dockerConfig) addEnv(envs ...string) {
	for _, env := range envs {
		pos := strings.Index(env, "=")
		var name string
		if pos > 0 {
			name = env[0:pos]
		} else {
			name = env
		}
		found := false
		for n, confEnv := range c.Envs {
			if name == confEnv || strings.HasPrefix(confEnv, name+"=") {
				c.Envs[n] = env
				found = true
				break
			}
		}
		if !found {
			c.Envs = append(c.Envs, env)
		}
	}
}

type dockerRunner struct {
	task *hm.Task
}

// Runner wraps docker implementation
func Runner(task *hm.Task) (hm.TaskResult, error) {
	r := &dockerRunner{task: task}
	conf, err := r.loadConfig()
	if err == nil && conf.Image == "" {
		// image not present, fallback to shell
		return shell.Runner(task)
	}
	if err = r.run(conf); err != nil {
		return hm.Failure, err
	}
	return hm.Success, nil
}

func (r *dockerRunner) loadConfig() (conf *dockerConfig, err error) {
	conf = &dockerConfig{}

	if err = r.task.Target.GetSettingWithExt(SettingName, conf); err != nil {
		return
	}
	if conf.SrcVolume == "" {
		conf.SrcVolume = DefaultSrcVolume
	}
	conf.addEnv(r.task.Target.Envs...)
	for name, value := range r.task.Plan.Env {
		conf.addEnv(name + "=" + value)
	}
	conf.addEnv("HMAKE_PROJECT_DIR=" + conf.SrcVolume)
	conf.addEnv("HMAKE_PROJECT_FILE=" +
		filepath.Join(conf.SrcVolume,
			filepath.Base(r.task.Project().MasterFile.Source)))
	conf.addEnv("HMAKE_WORK_DIR=" +
		filepath.Join(conf.SrcVolume,
			filepath.Base(r.task.Plan.WorkPath)))
	conf.addEnv("HMAKE_TARGET=" + r.task.Name())
	return
}

func (r *dockerRunner) run(conf *dockerConfig) error {
	project := r.task.Project()

	dockerRun := []string{"docker", "run",
		"-a", "STDOUT", "-a", "STDERR",
		"--rm",
		"-v", project.BaseDir + ":" + conf.SrcVolume,
		"-w", conf.SrcVolume,
		"--entrypoint", filepath.Join(conf.SrcVolume, hm.WorkFolder, filepath.Base(r.task.ScriptFile())),
	}
	// by default, use non-root user
	if conf.User == "" {
		dockerRun = append(dockerRun, "-u", strconv.Itoa(os.Getuid()))
	} else if conf.User != "root" && conf.User != "0" {
		dockerRun = append(dockerRun, "-u", conf.User)
	}

	for _, envFile := range conf.EnvFiles {
		dockerRun = append(dockerRun, "--env-file", envFile)
	}

	for _, env := range conf.Envs {
		dockerRun = append(dockerRun, "-e", env)
	}

	if conf.Network == "host" {
		dockerRun = append(dockerRun, "--net=host", "--uts=host")
	} else {
		for _, host := range conf.Hosts {
			dockerRun = append(dockerRun, "--add-host", host)
		}
		for _, dns := range conf.DNSServers {
			dockerRun = append(dockerRun, "--dns", dns)
		}
		if conf.DNSSearch != "" {
			dockerRun = append(dockerRun, "--dns-search", conf.DNSSearch)
		}
		for _, opt := range conf.DNSOpts {
			dockerRun = append(dockerRun, "--dns-opt", opt)
		}
	}

	for _, cap := range conf.CapAdd {
		dockerRun = append(dockerRun, "--cap-add", cap)
	}
	for _, cap := range conf.CapDrop {
		dockerRun = append(dockerRun, "--cap-drop", cap)
	}

	for _, dev := range conf.Devices {
		dockerRun = append(dockerRun, "--device", dev)
	}

	if conf.Privileged {
		dockerRun = append(dockerRun, "--privileged")
	}

	for _, vol := range conf.Volumes {
		dockerRun = append(dockerRun, "-v", vol)
	}

	if conf.BlkIoWeight != nil {
		dockerRun = append(dockerRun, "--blkio-weight", strconv.Itoa(*conf.BlkIoWeight))
	}
	for _, w := range conf.BlkIoWeightDevs {
		dockerRun = append(dockerRun, "--blkio-weight-device", w)
	}
	for _, bps := range conf.DevReadBps {
		dockerRun = append(dockerRun, "--device-read-bps", bps)
	}
	for _, bps := range conf.DevWriteBps {
		dockerRun = append(dockerRun, "--device-write-bps", bps)
	}
	for _, iops := range conf.DevReadIops {
		dockerRun = append(dockerRun, "--device-read-iops", iops)
	}
	for _, iops := range conf.DevWriteIops {
		dockerRun = append(dockerRun, "--device-write-iops", iops)
	}
	if conf.CPUShares != nil {
		dockerRun = append(dockerRun, "-c", strconv.Itoa(*conf.CPUShares))
	}
	if conf.CPUPeriod != nil {
		dockerRun = append(dockerRun, "--cpu-period", strconv.Itoa(*conf.CPUPeriod))
	}
	if conf.CPUQuota != nil {
		dockerRun = append(dockerRun, "--cpu-quota", strconv.Itoa(*conf.CPUQuota))
	}
	if conf.CPUSetCPUs != "" {
		dockerRun = append(dockerRun, "--cpuset-cpus", conf.CPUSetCPUs)
	}
	if conf.CPUSetMems != "" {
		dockerRun = append(dockerRun, "--cpuset-mems", conf.CPUSetMems)
	}
	if conf.KernelMemory != "" {
		dockerRun = append(dockerRun, "--kernel-memory", conf.KernelMemory)
	}
	if conf.Memory != "" {
		dockerRun = append(dockerRun, "-m", conf.Memory)
	}
	if conf.MemorySwap != "" {
		dockerRun = append(dockerRun, "--memory-swap", conf.MemorySwap)
	}
	if conf.MemorySwappiness != nil {
		dockerRun = append(dockerRun, "--memory-swappiness", strconv.Itoa(*conf.MemorySwappiness))
	}
	if conf.MemoryReservation != "" {
		dockerRun = append(dockerRun, "--memory-reservation", conf.MemoryReservation)
	}
	if conf.ShmSize != "" {
		dockerRun = append(dockerRun, "--shm-size", conf.ShmSize)
	}

	dockerRun = append(dockerRun, conf.Image)

	script, err := r.task.BuildScriptFile()
	if err != nil || script == "" {
		return err
	}

	return r.task.Exec(dockerRun[0], dockerRun[1:]...)
}

func init() {
	hm.RegisterExecDriver(ExecDriverName, Runner)
}
