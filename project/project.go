package project

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
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

func loadYaml(filename string) (map[string]interface{}, error) {
	data, err := ioutil.ReadFile(filename)
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
	case []map[interface{}]interface{}:
		a := make([]interface{}, len(v))
		for n, item := range v {
			a[n] = normalizeMap(item)
		}
		val = a
	case []map[string]interface{}:
		a := make([]interface{}, len(v))
		for n, item := range v {
			a[n] = normalizeMap(item)
		}
		val = a
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

// LoadFile loads from specified path
func LoadFile(baseDir, path string) (*File, error) {
	val, err := loadYaml(filepath.Join(baseDir, path))
	if err != nil {
		return nil, err
	}

	if format, ok := val["format"].(string); !ok || format != Format {
		return nil, fmt.Errorf("unsupported format: " + format)
	}

	f := &File{}
	err = mapper.Map(f, val)
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
	f, err := LoadFile(p.BaseDir, path)
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
