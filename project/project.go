package project

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/easeway/langx.go/errors"
	"github.com/easeway/langx.go/mapper"
	zglob "github.com/mattn/go-zglob"
	yaml "gopkg.in/yaml.v2"
)

const (
	// Format is the supported format
	Format = "hypermake.v0"
	// RcFile is the filename of local setting file to override some settings
	RcFile = ".hmakerc"
	// WorkFolder is the name of project WorkFolder
	WorkFolder = ".hmake"
	// SummaryFileName is the filename of summary
	SummaryFileName = "hmake.summary.json"
	// LogFileName is the filename of hmake debug log
	LogFileName = "hmake.debug.log"
	// WrapperMagic is the magic string at the beginning of the file
	WrapperMagic = "#hmake-wrapper"
	// WrapperName is project name for wrapped project
	WrapperName = "wrapper"
	// WrapperDesc is project description for wrapped project
	WrapperDesc = "wrapped HyperMake project"
	// MaxNameLen restricts the maximum length of project/target name
	MaxNameLen = 1024
)

var (
	// RootFile is hmake filename sits on root
	RootFile = "HyperMake"

	// ErrUnsupportedFormat indicates the file is not recognized
	ErrUnsupportedFormat = fmt.Errorf("unsupported format")
	// ErrNameMissing indicates name is required, but missing
	ErrNameMissing = fmt.Errorf("name is required")
	// ErrNameTooLong indicates the length of name exceeds MaxNameLen
	ErrNameTooLong = fmt.Errorf("name is too long")
	// ErrNameFirstChar indicates the first char in name is illegal
	ErrNameFirstChar = fmt.Errorf("name must start from a letter or an underscore")
	// ErrProjectNameMissing indicates project name is absent
	ErrProjectNameMissing = fmt.Errorf("project name is required")
	// ErrWrapperImageMissing indicates image name is missing
	ErrWrapperImageMissing = fmt.Errorf("image name missing after " + WrapperMagic)
)

// File defines the content of HyperMake or .hmake file
type File struct {
	// Format indicates file format
	Format string `json:"format"`
	// Name is name of the project
	Name string `json:"name"`
	// Desc is description of the project
	Desc string `json:"description"`
	// Targets are targets defined in current file
	Targets map[string]*Target `json:"targets"`
	// Settings are properties
	Settings Settings `json:"settings"`
	// Local are properties only applied to file
	Local Settings `json:"local"`
	// Includes are patterns for sourcing external files
	Includes []string `json:"includes"`

	// Source is the relative path to the project
	Source string `json:"-"`
	// WrapperTarget specifies the default target in wrapper mode
	WrapperTarget string `json:"-"`
}

// Project is the world view of hmake
type Project struct {
	// Name is name of the project
	Name string
	// BaseDir is the root directory of project
	BaseDir string
	// LaunchPath is relative path under BaseDir where hmake launches
	LaunchPath string
	// MasterFile is the file with everything merged
	MasterFile File

	// All loaded make files
	Files []*File

	// Tasks are built from resolved targets
	Targets TargetNameMap
}

// CommonSettings are well known settings
type CommonSettings struct {
	DefaultTargets []string `json:"default-targets"`
	ExecTarget     string   `json:"exec-target"`
	ExecShell      string   `json:"exec-shell"`
}

func loadAndRender(fn string) ([]byte, error) {
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}

	// TODO templatizing

	return data, nil
}

func loadYaml(filename string) (map[string]interface{}, error) {
	data, err := loadAndRender(filename)
	if err != nil {
		return nil, err
	}
	val := make(map[string]interface{})
	if err = yaml.Unmarshal(data, val); err != nil {
		if !os.IsNotExist(err) {
			err = fmt.Errorf("%s: %v", filename, err)
		}
		return nil, err
	}

	// normalize yaml by converting
	// map[interface{}]interface{} to map[string]interface{}
	return normalizeMap(val).(map[string]interface{}), nil
}

func normalizeMap(val interface{}) interface{} {
	switch v := val.(type) {
	case []interface{}:
		for n, item := range v {
			v[n] = normalizeMap(item)
		}
	case map[interface{}]interface{}:
		m := make(map[string]interface{})
		for key, value := range v {
			m[fmt.Sprintf("%v", key)] = normalizeMap(value)
		}
		val = m
	case map[string]interface{}:
		for key, value := range v {
			v[key] = normalizeMap(value)
		}
	}
	return val
}

func loadAsWrapper(fn string) (*File, error) {
	data, err := loadAndRender(fn)
	if err != nil {
		return nil, err
	}

	rd := bufio.NewReader(bytes.NewBuffer(data))
	lineBytes, err := rd.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	if !bytes.HasPrefix(lineBytes, []byte(WrapperMagic+" ")) {
		return nil, err
	}
	line := string(bytes.TrimSpace(lineBytes[len(WrapperMagic)+1:]))
	tokens := strings.Split(line, " ")
	if len(tokens) == 0 || tokens[0] == "" {
		return nil, ErrWrapperImageMissing
	}

	image := tokens[0]
	tokens = tokens[1:]

	projFile := &File{
		Format:   Format,
		Name:     WrapperName,
		Desc:     WrapperDesc,
		Targets:  make(map[string]*Target),
		Settings: make(Settings),
	}

	buildFrom := ""
	var buildArgs []string
	for n, token := range tokens {
		if token != "" {
			buildFrom = token
			buildArgs = tokens[n+1:]
			break
		}
	}

	if buildFrom != "" {
		t := &Target{
			Name:    "toolchain",
			Desc:    "build toolchain image",
			Watches: []string{buildFrom},
			Ext: map[string]interface{}{
				"image": image,
				"build": buildFrom,
			},
		}
		if len(buildArgs) > 0 {
			t.Ext["build-args"] = buildArgs
		}
		projFile.Targets[t.Name] = t
	}
	t := &Target{
		Name:   "build",
		Desc:   "wrapped build target",
		Always: true,
		Ext: map[string]interface{}{
			"image": image,
		},
	}
	content, err := ioutil.ReadAll(rd)
	if err != nil {
		return nil, err
	}
	content = bytes.TrimSpace(content)
	if len(content) > 0 {
		if bytes.HasPrefix(content, []byte("#!")) {
			t.Ext["script"] = string(content)
		} else {
			t.Ext["script"] = "#!/bin/sh\n" + string(content)
		}
	} else {
		t.Ext["cmds"] = []string{
			`make "$@"`,
		}
	}
	if buildFrom != "" {
		t.After = append(t.After, "toolchain")
	}
	projFile.Targets[t.Name] = t
	projFile.WrapperTarget = t.Name
	projFile.Settings["default-targets"] = []string{t.Name}
	projFile.Settings["exec-target"] = t.Name
	return projFile, nil
}

var expandableTargetPattern = regexp.MustCompile(`^(\w+):(([\w-\.]+,)*)([\w-\.]+)$`)

type expToken struct {
	name   string
	values []string
	text   string
}

func parseTarget(name string) (tokens []expToken, err error) {
	str := name
	for len(str) > 0 {
		pos := strings.Index(str, "[")
		if pos >= 0 {
			if pos > 0 {
				tokens = append(tokens, expToken{text: str[:pos]})
			}
			pos1 := strings.Index(str[pos:], "]")
			if pos1 <= 0 {
				return nil, fmt.Errorf("invalid expandable target name: %s: missing ]", name)
			}
			token := expToken{text: str[pos+1 : pos+pos1]}
			if !expandableTargetPattern.MatchString(token.text) {
				return nil, fmt.Errorf("invalid expandable target name: %s: bad format: %s", name, token.text)
			}
			str = str[pos+pos1+1:]

			pos = strings.Index(token.text, ":")
			token.name = token.text[:pos]
			token.values = strings.Split(token.text[pos+1:], ",")
			tokens = append(tokens, token)
		} else {
			tokens = append(tokens, expToken{text: str})
			break
		}
	}
	return
}

// substitute string containing "$[var]" to the value in vars
// there's no escape, except "$[$]" is substituted to "$"
// undefined vars are not substituted
func substString(vars map[string]string, val string) (res string) {
	forVar := false
	for len(val) > 0 {
		if forVar {
			if pos := strings.Index(val, "]"); pos > 0 {
				name := val[2:pos]
				if name == "$" {
					res += name
				} else if v, ok := vars[name]; ok {
					res += v
				} else {
					res += val[:pos+1]
				}
				val = val[pos+1:]
				forVar = false
				continue
			}
		} else if pos := strings.Index(val, "$["); pos >= 0 {
			res += val[:pos]
			val = val[pos:]
			forVar = true
			continue
		}
		res += val
		break
	}
	return
}

func substStrings(vars map[string]string, strs []string) []string {
	result := make([]string, len(strs))
	for n, str := range strs {
		result[n] = substString(vars, str)
	}
	return result
}

func substMap(vars map[string]string, m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return m
	}
	d := make(map[string]interface{})
	for k, v := range m {
		d[substString(vars, k)] = substInterface(vars, v)
	}
	return d
}

func substSlice(vars map[string]string, v []interface{}) []interface{} {
	result := make([]interface{}, len(v))
	for n, val := range v {
		result[n] = substInterface(vars, val)
	}
	return result
}

func substInterface(vars map[string]string, v interface{}) interface{} {
	switch val := v.(type) {
	case string:
		return substString(vars, val)
	case []interface{}:
		return substSlice(vars, val)
	case map[string]interface{}:
		return substMap(vars, val)
	default:
		return v
	}
}

func buildTarget(origin *Target, name string, vars map[string]string, result map[string]*Target) error {
	if result[name] != nil {
		return fmt.Errorf("target already exists: %s", name)
	}
	t := &Target{
		Name:       name,
		Desc:       substString(vars, origin.Desc),
		Before:     substStrings(vars, origin.Before),
		After:      substStrings(vars, origin.After),
		ExecDriver: substString(vars, origin.ExecDriver),
		WorkDir:    substString(vars, origin.WorkDir),
		Watches:    substStrings(vars, origin.Watches),
		Artifacts:  substStrings(vars, origin.Artifacts),
		Ext:        substMap(vars, origin.Ext),
		Always:     origin.Always,
	}
	result[name] = t
	return nil
}

func constructTargets(tokens []expToken, prefix string, n int,
	t *Target, vars map[string]string, result map[string]*Target) error {
	if n >= len(tokens) {
		return buildTarget(t, prefix, vars, result)
	}
	if name := tokens[n].name; name != "" {
		for _, val := range tokens[n].values {
			vars[name] = val
			if err := constructTargets(tokens, prefix+val, n+1, t, vars, result); err != nil {
				return err
			}
		}
	} else {
		return constructTargets(tokens, prefix+tokens[n].text, n+1, t, vars, result)
	}
	return nil
}

func expandTargets(origin map[string]*Target) (map[string]*Target, error) {
	keys := make([]string, 0, len(origin))
	for key := range origin {
		keys = append(keys, key)
	}

	result := make(map[string]*Target)
	for _, key := range keys {
		tokens, err := parseTarget(key)
		if err != nil {
			return nil, err
		}
		err = constructTargets(tokens, "", 0, origin[key], make(map[string]string), result)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// LoadFile loads from specified path
func LoadFile(baseDir, path string, allowWrapper bool) (*File, error) {
	fn := filepath.Join(baseDir, path)
	if allowWrapper {
		if f, err := loadAsWrapper(fn); err != nil || f != nil {
			return f, err
		}
	}

	val, err := loadYaml(fn)
	if err != nil {
		return nil, err
	}

	if format, ok := val["format"].(string); !ok || format != Format {
		return nil, fmt.Errorf("unsupported format: " + format)
	}

	f := &File{}
	err = mapper.Map(f, val)
	if err != nil {
		return nil, err
	}
	f.Targets, err = expandTargets(f.Targets)
	if err == nil {
		f.Source = path
	}
	return f, err
}

// LocateProjectFrom creates a project by locating the root file from startDir
func LocateProjectFrom(startDir, projectFile string) (*Project, error) {
	wd, err := filepath.Abs(startDir)
	if err != nil {
		return nil, err
	}
	launchPath := ""

	for {
		p := &Project{BaseDir: wd, LaunchPath: launchPath}
		_, err := p.Load(projectFile)
		if err == nil {
			return p, nil
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
		dir := filepath.Dir(wd)
		if dir == wd {
			break
		}
		launchPath = filepath.Join(filepath.Base(wd), launchPath)
		wd = dir
	}

	return nil, os.ErrNotExist
}

// LocateProject creates a project by locating the root file
func LocateProject() (*Project, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return LocateProjectFrom(wd, RootFile)
}

// LoadProjectFrom locates, resolves and finalizes project from startDir
func LoadProjectFrom(startDir, projectFile string) (p *Project, err error) {
	if p, err = LocateProjectFrom(startDir, projectFile); err != nil {
		return
	}
	if err = p.Resolve(); err != nil {
		return
	}
	err = p.Finalize()
	return
}

// LoadProject locates, resolves and finalizes project
func LoadProject() (p *Project, err error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return LoadProjectFrom(wd, RootFile)
}

// RelPath translate a source relative path to project relative path
func RelPath(source, path string) string {
	srcDir := filepath.Dir(source)
	if srcDir != "." {
		return filepath.Join(srcDir, path)
	}
	return path
}

// ValidateName checks if a name is legal: starting from a letter/underscore
// and following characters are from letters/digits/underscore/dash/dot
func ValidateName(name string) error {
	if name == "" {
		return ErrNameMissing
	}
	if len(name) > MaxNameLen {
		return ErrNameTooLong
	}
	for n, r := range name {
		if n == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return ErrNameFirstChar
			}
		} else if !unicode.IsLetter(r) && !unicode.IsDigit(r) &&
			r != '_' && r != '-' && r != '.' {
			return fmt.Errorf("invalid character in name %v", r)
		}
	}
	return nil
}

// ValidateProjectName validates if project name is legal
func ValidateProjectName(name string) error {
	if name == "" {
		return ErrProjectNameMissing
	}
	return ValidateName(name)
}

// Merge merges content from another file
func (f *File) Merge(s *File) error {
	errs := &errors.AggregatedError{}
	if f.Targets == nil {
		f.Targets = make(map[string]*Target)
	}
	for name, t := range s.Targets {
		if target, exist := f.Targets[name]; exist {
			errs.Add(fmt.Errorf("duplicated target %s defined in %s and %s",
				name, target.File.Source, s.Source))
		} else {
			f.Targets[name] = t
		}
	}
	if f.Settings == nil {
		f.Settings = make(Settings)
	}
	errs.Add(f.Settings.Merge(s.Settings))

	for _, inc := range s.Includes {
		path := RelPath(s.Source, inc)
		found := false
		for _, item := range f.Includes {
			if item == path {
				found = true
				break
			}
		}
		if !found {
			f.Includes = append(f.Includes, path)
		}
	}
	return errs.Aggregate()
}

// Load loads and merges an additional files
func (p *Project) Load(path string) (*File, error) {
	for _, f := range p.Files {
		if f.Source == path {
			return f, nil
		}
	}
	f, err := LoadFile(p.BaseDir, path, len(p.Files) == 0)
	if err != nil {
		return nil, err
	}
	for name, t := range f.Targets {
		if err = ValidateName(name); err != nil {
			return nil, fmt.Errorf("%s: illegal target name '%s': %v",
				f.Source, name, err.Error())
		}
		t.File = f
	}
	p.Files = append(p.Files, f)
	if err = p.MasterFile.Merge(f); err != nil {
		return f, err
	}
	if len(p.Files) == 1 {
		p.MasterFile.Source = f.Source
		p.MasterFile.WrapperTarget = f.WrapperTarget
		p.Name = f.Name
		err = ValidateProjectName(p.Name)
	}
	return f, err
}

// LoadRcFiles load .hmakerc files inside project directories
func (p *Project) LoadRcFiles() error {
	var files []string
	path := p.LaunchPath
	for {
		files = append(files, filepath.Join(path, RcFile))
		if path == "" {
			break
		}
		dir := filepath.Dir(path)
		if dir == "." {
			path = ""
		} else {
			path = dir
		}
	}

	errs := &errors.AggregatedError{}
	for i := len(files) - 1; i >= 0; i-- {
		_, err := p.Load(files[i])
		if err != nil && !os.IsNotExist(err) {
			errs.Add(err)
		}
	}
	return errs.Aggregate()
}

// Glob matches files inside project with pattern
func (p *Project) Glob(pattern string) (paths []string, err error) {
	prefix := p.BaseDir + string(filepath.Separator)
	fullPattern := prefix + pattern
	paths, err = zglob.Glob(fullPattern)
	if err != nil {
		return
	}
	prefixLen := len(prefix)
	for n, fullpath := range paths {
		paths[n] = fullpath[prefixLen:]
	}
	return
}

// Resolve loads additional includes
func (p *Project) Resolve() error {
	errs := &errors.AggregatedError{}
	for i := 0; i < len(p.MasterFile.Includes); i++ {
		paths, err := p.Glob(p.MasterFile.Includes[i])
		if errs.Add(err) {
			continue
		}
		for _, path := range paths {
			st, err := os.Stat(filepath.Join(p.BaseDir, path))
			if errs.Add(err) || st.IsDir() {
				continue
			}
			_, err = p.Load(path)
			errs.Add(err)
		}
	}
	return errs.Aggregate()
}

// Finalize builds up the relationship between targets and settings
// and also verifies any cyclic dependencies
func (p *Project) Finalize() error {
	errs := errors.AggregatedError{}
	p.Targets = make(TargetNameMap)
	for name, t := range p.MasterFile.Targets {
		t.Initialize(name, p)
		errs.Add(p.Targets.Add(t))
	}
	errs.AddMany(
		p.Targets.BuildDeps(),
		p.Targets.CheckCyclicDeps(),
	)

	return errs.Aggregate()
}

// Plan creates an ExecPlan for this project
func (p *Project) Plan() *ExecPlan {
	return NewExecPlan(p)
}

// TargetNames returns sorted target names
func (p *Project) TargetNames() []string {
	targets := make([]string, 0, len(p.Targets))
	for name := range p.Targets {
		targets = append(targets, name)
	}
	sort.Strings(targets)
	return targets
}

// TargetNamesMatch returns sorted and matched target names
func (p *Project) TargetNamesMatch(pattern string) (names []string, err error) {
	names, err = p.Targets.CompleteName(pattern)
	if err == nil {
		sort.Strings(names)
	}
	return
}

// GetSettings maps settings into provided variable
func (p *Project) GetSettings(v interface{}) error {
	return p.MasterFile.Settings.Get(v)
}

// GetSettingsIn maps settings[name] into provided variable
func (p *Project) GetSettingsIn(name string, v interface{}) error {
	return p.MasterFile.Settings.GetBy(name, v)
}

// MergeSettingsFlat merges settings from a flat key/value map
func (p *Project) MergeSettingsFlat(flat map[string]interface{}) error {
	sets := p.MasterFile.Settings
	if sets == nil {
		sets = make(Settings)
		p.MasterFile.Settings = sets
	}
	return sets.MergeFlat(flat)
}

// WorkPath returns the internal state folder (.hmake) for hmake
func (p *Project) WorkPath() string {
	return filepath.Join(p.BaseDir, WorkFolder)
}

// DebugLogFile returns the fullpath to debug log file
func (p *Project) DebugLogFile() string {
	return filepath.Join(p.WorkPath(), LogFileName)
}

// SummaryFile returns the fullpath to summary file
func (p *Project) SummaryFile() string {
	return filepath.Join(p.WorkPath(), SummaryFileName)
}

// Summary loads the execution summary
func (p *Project) Summary() (ExecSummary, error) {
	f, err := os.Open(p.SummaryFile())
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var summary ExecSummary
	return summary, json.NewDecoder(f).Decode(&summary)
}

// WrapperTarget returns the wrapper target if in wrapper mode
func (p *Project) WrapperTarget() *Target {
	name := p.MasterFile.WrapperTarget
	if name != "" {
		return p.Targets[name]
	}
	return nil
}
