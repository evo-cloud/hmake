package docker

import (
	"archive/tar"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/easeway/langx.go/mapper"
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
	BuildArgs         []string `json:"build-args"`
	Commits           []string `json:"commit"`
	Push              []string `json:"push"`
	Tags              []string `json:"tags"`
	Labels            []string `json:"labels"`
	LabelFiles        []string `json:"label-files"`
	ForceRm           bool     `json:"force-rm"`
	Pull              bool     `json:"pull"`
	Cache             *bool    `json:"cache"`
	ContentTrust      *bool    `json:"content-trust"`
	Image             string   `json:"image"`
	SrcVolume         string   `json:"src-volume"`
	ExposeDocker      bool     `json:"expose-docker"`
	Env               []string `json:"env"`
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
	ULimit            []string `json:"ulimit"`

	// reserved properties
	NoPasswdPatch bool `json:"no-passwd-patch"`

	projectDir string
}

func (r *Runner) logf(format string, args ...interface{}) {
	r.Task.Plan.Logf(format, args...)
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
		for n, confEnv := range r.Env {
			if name == confEnv || strings.HasPrefix(confEnv, name+"=") {
				r.Env[n] = env
				found = true
				break
			}
		}
		if !found {
			r.Env = append(r.Env, env)
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

func (r *Runner) exec(args ...string) *shell.Executor {
	x := shell.Exec(r.Task, "docker", args...)
	// env are passed with -e, no need for docker client
	x.Cmd.Env = os.Environ()
	return x
}

func (r *Runner) docker(args ...string) error {
	return r.exec(args...).Mute().Run(nil)
}

func (r *Runner) dockerPiped(in io.Reader, out io.Writer, sigCh <-chan os.Signal, args ...string) error {
	x := r.exec(args...).Mute()
	x.Cmd.Stdin = in
	x.Cmd.Stdout = out
	var errOut bytes.Buffer
	x.Cmd.Stderr = &errOut
	err := x.Run(sigCh)
	if err != nil {
		r.logf("docker piped stderr: %v: %s", err, errOut.String())
	}
	return err
}

func (r *Runner) signal(sig os.Signal, relayCh chan os.Signal) {
	sysSig := sig.(syscall.Signal)
	if cid := r.cid(); cid != "" {
		r.logf("Signal container %s %d", cid, sysSig)
		// HACK: in non-tty mode, docker is not going to pass the signal to init
		// process, then the INTERRUPT/TERM signal should be translated into kill
		if sysSig == syscall.SIGINT || sysSig == syscall.SIGTERM {
			r.docker("kill", cid)
		} else {
			r.docker("kill", "-s", strconv.Itoa(int(sysSig)), cid)
		}
	} else {
		// CID not available, probably the image is being downloaded
		// send the signal to docker client
		r.logf("Relay signal %d, CID not available", sysSig)
		relayCh <- sig
	}
}

func (r *Runner) removeContainer() {
	if cid := r.cid(); cid != "" {
		r.logf("Removing container %s", cid)
		r.docker("rm", "-f", cid)
	} else {
		r.logf("Ignore removing container, CID not available")
	}
}

func (r *Runner) commonOpts(args *shell.Args) {
	if r.CPUShares != nil {
		args.Add("--cpu-shares", strconv.Itoa(*r.CPUShares))
	}
	if r.CPUPeriod != nil {
		args.Add("--cpu-period", strconv.Itoa(*r.CPUPeriod))
	}
	if r.CPUQuota != nil {
		args.Add("--cpu-quota", strconv.Itoa(*r.CPUQuota))
	}
	if r.CPUSetCPUs != "" {
		args.Add("--cpuset-cpus", r.CPUSetCPUs)
	}
	if r.CPUSetMems != "" {
		args.Add("--cpuset-mems", r.CPUSetMems)
	}
	if r.Memory != "" {
		args.Add("-m", r.Memory)
	}
	if r.MemorySwap != "" {
		args.Add("--memory-swap", r.MemorySwap)
	}
	if r.ShmSize != "" {
		args.Add("--shm-size", r.ShmSize)
	}
	for _, lim := range r.ULimit {
		args.Add("--ulimit", lim)
	}
	for _, label := range r.Labels {
		args.Add("--label", label)
	}
	for _, labelFile := range r.LabelFiles {
		args.Add("--label-file", labelFile)
	}
	if r.ContentTrust != nil && !*r.ContentTrust {
		args.Add("--disable-content-trust")
	}
}

// Run implements Runner
func (r *Runner) Run(sigCh <-chan os.Signal) (result hm.TaskResult, err error) {
	result = hm.Success

	os.Remove(r.cidFile())
	if r.Task.Target.Exec {
		err = r.run(sigCh)
	} else {
		if r.Build != "" {
			err = r.build(sigCh)
		}
		if err == nil {
			err = r.run(sigCh)
		}
		if err == nil && len(r.Commits) > 0 {
			err = r.commit(sigCh)
		}
		if err == nil && len(r.Push) > 0 {
			err = r.push(sigCh)
		}
	}
	if err != nil {
		result = hm.Failure
	}
	r.removeContainer()
	return
}

func (r *Runner) build(sigCh <-chan os.Signal) error {
	dockerCmd := shell.NewArgs("build", "-t", r.Image)

	for _, arg := range r.BuildArgs {
		dockerCmd.Add("--build-arg", arg)
	}
	for _, tag := range r.Tags {
		dockerCmd.Add("-t", tag)
	}
	if r.ForceRm {
		dockerCmd.Add("--force-rm")
	}
	if r.Pull {
		dockerCmd.Add("--pull")
	}
	if r.Cache != nil && !*r.Cache {
		dockerCmd.Add("--no-cache")
	}

	r.commonOpts(dockerCmd)

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
			dockerCmd.Add(dockerFile)
		} else {
			dockerCmd.Add("-f", filepath.Join(dockerFile, Dockerfile), buildFrom)
		}
	} else if buildFrom == "" {
		dockerCmd.Add("-f", dockerFile, filepath.Dir(dockerFile))
	} else {
		dockerCmd.Add("-f", dockerFile, buildFrom)
	}

	return r.exec(dockerCmd.Args...).Run(sigCh)
}

func (r *Runner) commit(sigCh <-chan os.Signal) error {
	imageName := r.Commits[0]
	commitCmd := shell.NewArgs("commit", r.cid(), imageName)
	err := r.exec(commitCmd.Args...).Run(sigCh)
	if err != nil {
		return err
	}
	for i := 1; i < len(r.Commits); i++ {
		tagCmd := shell.NewArgs("tag", imageName, r.Commits[i])
		if err = r.exec(tagCmd.Args...).Run(sigCh); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) push(sigCh <-chan os.Signal) error {
	for _, img := range r.Push {
		cmd := shell.NewArgs("push", img)
		if err := r.exec(cmd.Args...).Run(sigCh); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) run(sigCh <-chan os.Signal) error {
	err := r.checkProjectDir()
	if err != nil {
		return err
	}

	workDir := filepath.Join(r.SrcVolume, r.Task.Target.WorkingDir())
	dockerCmd := shell.NewArgs("create",
		"-v", r.canonicalProjectDir()+":"+r.SrcVolume,
		"-w", filepath.ToSlash(workDir),
		"--cidfile", r.cidFile(),
	)

	execArgs := r.Task.Target.Args
	dockerCmd.Add("--entrypoint")
	if r.Task.Target.Exec {
		if len(execArgs) > 0 {
			dockerCmd.Add(execArgs[0])
			execArgs = execArgs[1:]
		} else {
			settings, e := r.Task.Target.CommonSettings()
			if e != nil {
				return e
			}
			if settings.ExecShell == "" {
				settings.ExecShell = "/bin/sh"
			}
			dockerCmd.Add(settings.ExecShell)
		}
	} else {
		dockerCmd.Add(filepath.ToSlash(filepath.Join(r.SrcVolume, hm.WorkFolder,
			filepath.Base(shell.ScriptFile(r.Task)))))
	}

	// support console
	var shellTarget shell.Target
	r.Task.Target.GetExt(&shellTarget)
	console := r.Task.Target.Exec || shellTarget.Console
	if console {
		dockerCmd.Add("-it")
	} else {
		dockerCmd.Add("-a", "STDOUT", "-a", "STDERR")
	}

	var passwd passwdPatcher

	// by default, use non-root user
	if r.User == "" {
		if e := passwd.current(); e != nil {
			return e
		}
		dockerCmd.Add("-u", passwd.user())
		if len(r.Groups) == 0 {
			for _, grp := range passwd.groups {
				if grp != passwd.gid {
					dockerCmd.Add("--group-add", strconv.Itoa(grp))
				}
			}
		}
	} else if r.User != "root" && r.User != "0" {
		if e := passwd.parse(r.User); e != nil {
			return e
		}
		dockerCmd.Add("-u", passwd.user())
	}
	for _, grp := range r.Groups {
		passwd.addGroup(grp)
		dockerCmd.Add("--group-add", grp)
	}

	for _, envFile := range r.EnvFiles {
		dockerCmd.Add("--env-file",
			filepath.Join(r.SrcVolume, r.Task.Target.WorkingDir(envFile)))
	}

	for _, env := range r.Env {
		dockerCmd.Add("-e", env)
	}

	if r.Network == "host" {
		dockerCmd.Add("--net=host", "--uts=host")
	} else {
		for _, host := range r.Hosts {
			dockerCmd.Add("--add-host", host)
		}
		for _, dns := range r.DNSServers {
			dockerCmd.Add("--dns", dns)
		}
		if r.DNSSearch != "" {
			dockerCmd.Add("--dns-search", r.DNSSearch)
		}
		for _, opt := range r.DNSOpts {
			dockerCmd.Add("--dns-opt", opt)
		}
	}

	for _, cap := range r.CapAdd {
		dockerCmd.Add("--cap-add", cap)
	}
	for _, cap := range r.CapDrop {
		dockerCmd.Add("--cap-drop", cap)
	}

	for _, dev := range r.Devices {
		dockerCmd.Add("--device", dev)
	}

	if r.Privileged {
		dockerCmd.Add("--privileged")
	}

	for _, vol := range r.Volumes {
		hostVol := vol
		if strings.HasPrefix(hostVol, "~/") {
			hostVol = filepath.Join(os.Getenv("HOME"), hostVol[2:])
		} else if strings.HasPrefix(hostVol, "-/") {
			hostVol = filepath.Join(r.projectDir, hostVol[2:])
		} else if !filepath.IsAbs(hostVol) {
			hostVol = filepath.Join(r.projectDir, r.Task.Target.WorkingDir(vol))
		}
		dockerCmd.Add("-v", hostVol)
	}

	if r.BlkIoWeight != nil {
		dockerCmd.Add("--blkio-weight", strconv.Itoa(*r.BlkIoWeight))
	}
	for _, w := range r.BlkIoWeightDevs {
		dockerCmd.Add("--blkio-weight-device", w)
	}
	for _, bps := range r.DevReadBps {
		dockerCmd.Add("--device-read-bps", bps)
	}
	for _, bps := range r.DevWriteBps {
		dockerCmd.Add("--device-write-bps", bps)
	}
	for _, iops := range r.DevReadIops {
		dockerCmd.Add("--device-read-iops", iops)
	}
	for _, iops := range r.DevWriteIops {
		dockerCmd.Add("--device-write-iops", iops)
	}

	r.commonOpts(dockerCmd)

	if r.KernelMemory != "" {
		dockerCmd.Add("--kernel-memory", r.KernelMemory)
	}
	if r.MemorySwappiness != nil {
		dockerCmd.Add("--memory-swappiness", strconv.Itoa(*r.MemorySwappiness))
	}
	if r.MemoryReservation != "" {
		dockerCmd.Add("--memory-reservation", r.MemoryReservation)
	}

	dockerCmd.Add(r.Image)

	if r.Task.Target.Exec {
		dockerCmd.Add(execArgs...)
	} else {
		script, e := shell.BuildScriptFile(r.Task)
		if e != nil || script == "" {
			return e
		}
	}

	// create container
	if err = r.exec(dockerCmd.Args...).MuteOut().Run(sigCh); err != nil {
		return err
	}

	if !r.NoPasswdPatch {
		if err = passwd.patch(r, sigCh); err != nil {
			return err
		}
	}

	dockerCmd = shell.NewArgs("start", "-a")
	if console {
		dockerCmd.Add("-i")
	}
	dockerCmd.Add(r.cid())

	x := r.exec(dockerCmd.Args...)

	if console {
		// tty mode, CtrlC is handled by docker client
		err = x.Run(sigCh)
	} else {
		// non-tty mode, CtrlC is not handled properly
		ch := make(chan struct{})
		sigRelay := make(chan os.Signal, 1)
		go func() {
			for {
				select {
				case <-ch:
					return
				case sig := <-sigCh:
					r.signal(sig, sigRelay)
				}
			}
		}()
		err = x.Run(sigRelay)
		close(ch)
	}
	return err
}

func sortStrs(src []string) []string {
	dst := make([]string, len(src))
	copy(dst, src)
	sort.Strings(dst)
	return dst
}

// Signature implements Runner
func (r *Runner) Signature() string {
	dict := make(map[string]interface{})
	err := mapper.Map(dict, r)
	if err != nil {
		panic(err)
	}
	keys := make([]string, 0, len(dict))
	for k, v := range dict {
		keys = append(keys, k)
		switch k {
		case "commit", "push", "tags", "labels", "label-files",
			"cap-add", "cap-drop", "devices":
			dict[k] = sortStrs(v.([]string))
		case "env":
			// environment variables need special handling
			// skip environment variables which changes but
			// should not affect signature
			sorted := sortStrs(v.([]string))
			vars := make([]string, 0, len(sorted))
			for _, item := range sorted {
				if strings.HasPrefix(item, "HMAKE_") {
					continue
				}
				vars = append(vars, item)
			}
			dict[k] = vars
		}
	}
	sort.Strings(keys)
	items := make([]string, len(keys))
	for n, k := range keys {
		item := k + "="
		val := dict[k]
		switch v := val.(type) {
		case []string:
			item += "[" + strings.Join(v, ",") + "]"
		case *bool:
			if v != nil {
				item += fmt.Sprintf("%v", v)
			}
		case *int:
			if v != nil {
				item += fmt.Sprintf("%v", v)
			}
		default:
			item += fmt.Sprintf("%v", v)
		}
		items[n] = item
	}
	return strings.Join(items, ",") + "\n" + shell.BuildScript(r.Task)
}

// ValidateArtifacts implements Runner
func (r *Runner) ValidateArtifacts() bool {
	var images []string
	if r.Build != "" || r.BuildFrom != "" {
		images = append(images, r.Image)
		if len(r.Tags) > 0 {
			images = append(images, r.Tags...)
		}
	}
	if len(r.Commits) > 0 {
		images = append(images, r.Commits...)
	}

	for _, image := range images {
		if err := r.docker("inspect", "-f", "{{.Id}}", image); err != nil {
			r.logf("docker artifact invalid: %s: %v", image, err)
			return false
		}
		r.logf("docker artifact ok: %s", image)
	}
	return true
}

// Factory is runner factory
func Factory(task *hm.Task) (hm.Runner, error) {
	r := &Runner{Task: task}

	if err := task.Target.GetSettingsWithExt(SettingName, r); err != nil {
		return nil, err
	}

	if r.Image == "" {
		return nil, fmt.Errorf("missing property image")
	}

	if r.SrcVolume == "" {
		r.SrcVolume = DefaultSrcVolume
	}
	if r.ExposeDocker {
		r.exposeDocker()
	}
	for name, value := range task.Plan.Env {
		r.addEnv(name + "=" + value)
	}
	r.addEnv("HMAKE_PROJECT_DIR=" + r.SrcVolume)
	r.addEnv("HMAKE_PROJECT_FILE=" +
		filepath.ToSlash(filepath.Join(r.SrcVolume,
			filepath.Base(task.Project().MasterFile.Source))))
	r.addEnv("HMAKE_WORK_DIR=" +
		filepath.Join(r.SrcVolume,
			filepath.Base(task.Plan.WorkPath)))
	r.addEnv(task.EnvVars()...)

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
	r.addEnv("HMAKE_DOCKER_VOL_HOST=" + r.canonicalProjectDir())
	r.addEnv("HMAKE_DOCKER_VOL_CNTR=" + r.SrcVolume)
	return r, nil
}

type passwdPatcher struct {
	uid, gid int
	groups   []int
}

func (p *passwdPatcher) current() (err error) {
	p.uid, p.gid, p.groups, err = currentUserIds()
	return
}

func (p *passwdPatcher) user() string {
	return strconv.Itoa(p.uid) + ":" + strconv.Itoa(p.gid)
}

func (p *passwdPatcher) parse(user string) (err error) {
	p.uid, p.gid, err = userID(user)
	return
}

func (p *passwdPatcher) addGroup(grp string) {
	if gid, err := strconv.Atoi(grp); err == nil {
		p.groups = append(p.groups, gid)
	}
}

func (p *passwdPatcher) patch(r *Runner, sigCh <-chan os.Signal) (err error) {
	if p.uid == 0 {
		// no need
		return
	}
	uidStr := strconv.Itoa(p.uid)

	var out bytes.Buffer
	err = r.dockerPiped(nil, &out, sigCh, "cp", r.cid()+":/etc/passwd", "-")
	if err != nil {
		return
	}
	tarRd := tar.NewReader(bytes.NewBuffer(out.Bytes()))
	header, err := tarRd.Next()
	if err != nil {
		if err == io.EOF {
			// TODO skip?
			err = nil
		}
		r.logf("untar /etc/passwd error: %v", err)
		return
	}

	var lines []string
	scanner := bufio.NewScanner(tarRd)
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, ":")
		if len(tokens) >= 3 && tokens[2] == uidStr {
			// already exist, no need to patch
			return
		}
		lines = append(lines, line)
	}
	if err = scanner.Err(); err != nil {
		r.logf("scan /etc/passwd error: %v", err)
		return
	}

	lines = append(lines, fmt.Sprintf("user%d:x:%d:%d::/tmp:/sbin/nologin", p.uid, p.uid, p.gid))
	content := []byte(strings.Join(lines, "\n"))

	var gen bytes.Buffer
	w := tar.NewWriter(&gen)
	header.Size = int64(len(content))
	if err = w.WriteHeader(header); err != nil {
		r.logf("write back tar header error: %v", err)
		return
	}
	if _, err = w.Write(content); err != nil {
		r.logf("write back tar content error: %v", err)
		return
	}
	w.Close()

	err = r.dockerPiped(bytes.NewBuffer(gen.Bytes()), nil, sigCh, "cp", "-", r.cid()+":/etc")

	return
}

func currentUserIds() (uid, gid int, grps []int, err error) {
	if usingDockerMachine() {
		return currentUserIdsFromDockerMachine()
	}
	uid = os.Getuid()
	gid = os.Getgid()
	grps, err = os.Getgroups()
	return
}

func userID(name string) (uid, gid int, err error) {
	if usingDockerMachine() {
		return userIDFromDockerMachine(name)
	}
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

func init() {
	hm.RegisterExecDriver(ExecDriverName, Factory)
}
