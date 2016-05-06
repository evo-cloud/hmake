package project

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/easeway/langx.go/errors"
	"github.com/easeway/langx.go/mapper"
	zglob "github.com/mattn/go-zglob"
	yaml "gopkg.in/yaml.v2"
)

const (
	// Format is the supported format
	Format = "hypermake.v0"
	// RootFile is hmake filename sits on root
	RootFile = "HyperMake"
	// WorkFolder is the name of project WorkFolder
	WorkFolder = ".hmake"
	// SummaryFileName is the filename of summary
	SummaryFileName = "hmake.summary.json"
	// LogFileName is the filename of hmake debug log
	LogFileName = "hmake.debug.log"
)

// ErrUnsupportedFormat indicates the file is not recognized
var ErrUnsupportedFormat = fmt.Errorf("unsupported format")

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
	// Includes are patterns for sourcing external files
	Includes []string `json:"includes"`

	// Source is the relative path to the file
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
func LocateProjectFrom(startDir string) (*Project, error) {
	wd, err := filepath.Abs(startDir)
	if err != nil {
		return nil, err
	}
	launchPath := ""

	for {
		p := &Project{BaseDir: wd, LaunchPath: launchPath}
		_, err := p.Load(RootFile)
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
	return LocateProjectFrom(wd)
}

// LoadProjectFrom locates, resolves and finalizes project from startDir
func LoadProjectFrom(startDir string) (p *Project, err error) {
	if p, err = LocateProjectFrom(startDir); err != nil {
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
	return LoadProjectFrom(wd)
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
				name, target.Source, t.Source))
		} else {
			f.Targets[name] = t
		}
	}
	if f.Settings == nil {
		f.Settings = make(Settings)
	}
	errs.Add(f.Settings.Merge(s.Settings))

	for _, inc := range s.Includes {
		found := false
		for _, item := range f.Includes {
			if item == inc {
				found = true
				break
			}
		}
		if !found {
			f.Includes = append(f.Includes, inc)
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
	for _, t := range f.Targets {
		t.Source = f.Source
	}
	p.Files = append(p.Files, f)
	if err = p.MasterFile.Merge(f); err != nil {
		return f, err
	}
	if len(p.Files) == 1 {
		p.MasterFile.Source = f.Source
		p.Name = f.Name
	}
	return f, nil
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

// GetSettings maps settings into provided variable
func (p *Project) GetSettings(v interface{}) error {
	if p.MasterFile.Settings != nil {
		return mapper.Map(v, p.MasterFile.Settings)
	}
	return nil
}

// GetSettingsIn maps settings[name] into provided variable
func (p *Project) GetSettingsIn(name string, v interface{}) error {
	if p.MasterFile.Settings == nil {
		return nil
	}
	if val, exists := p.MasterFile.Settings[name]; exists {
		return mapper.Map(v, val)
	}
	return nil
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
