package project

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/easeway/langx.go/errors"
	"github.com/easeway/langx.go/mapper"
)

// Target defines a build target
type Target struct {
	Name       string                 `json:"name"`
	Desc       string                 `json:"description"`
	Before     []string               `json:"before"`
	After      []string               `json:"after"`
	ExecDriver string                 `json:"exec-driver"`
	WorkDir    string                 `json:"workdir"`
	Envs       []string               `json:"envs"`
	Cmds       []*Command             `json:"cmds"`
	Script     string                 `json:"script"`
	Watches    []string               `json:"watches"`
	Ext        map[string]interface{} `json:"*"`

	// Runtime fields

	Project   *Project      `json:"-"`
	Source    string        `json:"-"`
	Depends   TargetNameMap `json:"-"`
	Activates TargetNameMap `json:"-"`
}

// TargetNameMap is targets mapping by name
type TargetNameMap map[string]*Target

// Command defines a single command to execute
type Command struct {
	Shell string                 `json:"*"`
	Ext   map[string]interface{} `json:"*"`
}

// Settings applies to targets
type Settings map[string]interface{}

// WatchItem contains information the target watches
type WatchItem struct {
	// Path is relative path to the project root
	Path string
	// ModTime is the modification time of the item
	ModTime time.Time
}

// WatchList is list of watched items
type WatchList []*WatchItem

// Initialize prepare fields in target
func (t *Target) Initialize(name string, project *Project) {
	t.Name = name
	t.Project = project
	t.Depends = make(TargetNameMap)
	t.Activates = make(TargetNameMap)
}

// GetExt maps Ext to provided value
func (t *Target) GetExt(v interface{}) error {
	if t.Ext != nil {
		return mapper.Map(v, t.Ext)
	}
	return nil
}

// Errorf formats an error related to the target
func (t *Target) Errorf(format string, args ...interface{}) error {
	args = append([]interface{}{t.Name, t.Source}, args...)
	return fmt.Errorf("%s(%s): "+format, args...)
}

// AddDep adds a dependency with corresponding activates
func (t *Target) AddDep(dep *Target) {
	t.Depends[dep.Name] = dep
	dep.Activates[t.Name] = t
}

// GetSettings extracts the value from settings stack
func (t *Target) GetSettings(name string, v interface{}) error {
	return t.Project.GetSettingsIn(name, v)
}

// GetSettingsWithExt extracts the value from Ext and settings stack
func (t *Target) GetSettingsWithExt(name string, v interface{}) (err error) {
	if err = t.GetSettings(name, v); err == nil && t.Ext != nil {
		err = mapper.Map(v, t.Ext)
	}
	return
}

// BuildWatchList collects current state of all watched items
func (t *Target) BuildWatchList() (list WatchList) {
	files := make(map[string]*WatchItem)
	excludes := make(map[string]*WatchItem)
	for _, pattern := range t.Watches {
		dict := files
		if strings.HasPrefix(pattern, "!") {
			dict = excludes
			pattern = pattern[1:]
		}
		paths, err := t.Project.Glob(filepath.Join(filepath.Dir(t.Source), pattern))
		if err != nil {
			continue
		}
		for _, path := range paths {
			fullpath := filepath.Join(t.Project.BaseDir, path)
			st, err := os.Stat(fullpath)
			if err != nil {
				continue
			}
			if st.IsDir() {
				filepath.Walk(fullpath, func(relpath string, st os.FileInfo, err error) error {
					if err == nil {
						relpath = path + relpath[len(fullpath):]
						if !st.IsDir() {
							dict[relpath] = &WatchItem{Path: relpath, ModTime: st.ModTime()}
						}
					}
					return nil
				})
			} else {
				dict[path] = &WatchItem{Path: path, ModTime: st.ModTime()}
			}
		}
	}

	for path := range excludes {
		delete(files, path)
	}

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		list = append(list, files[name])
	}
	return
}

// Add adds a target to name map
func (m TargetNameMap) Add(t *Target) error {
	if target, exists := m[t.Name]; exists {
		return t.Errorf("target name duplicated in %s", target.Source)
	}
	m[t.Name] = t
	return nil
}

// BuildDeps builds direct depends and activates
func (m TargetNameMap) BuildDeps() error {
	errs := &errors.AggregatedError{}
	for _, t := range m {
		// convert before to after in target
		for _, name := range t.Before {
			dest, ok := m[name]
			if !ok {
				errs.Add(t.Errorf("before %s which is not defined", name))
			} else {
				dest.AddDep(t)
			}
		}
		// add depends for all after
		for _, name := range t.After {
			dest, ok := m[name]
			if !ok {
				errs.Add(t.Errorf("after %s which is not defined", name))
			} else {
				t.AddDep(dest)
			}
		}
	}
	return errs.Aggregate()
}

// CheckCyclicDeps detects cycles in depenencies
func (m TargetNameMap) CheckCyclicDeps() error {
	errs := &errors.AggregatedError{}
	unresolved := make(TargetNameMap)
	allDeps := make(map[string]TargetNameMap)
	// build direct dependencies
	for _, t := range m {
		unresolved[t.Name] = t
		deps := make(TargetNameMap)
		allDeps[t.Name] = deps
		for name, dep := range t.Depends {
			deps[name] = dep
		}
	}
	// merge indirect dependencies
	for len(unresolved) > 0 {
		var t *Target
		for _, t = range unresolved {
			break
		}
		m.resolveDeps(t, unresolved, allDeps, errs)
	}
	return errs.Aggregate()
}

func (m TargetNameMap) resolveDeps(t *Target,
	unresolved TargetNameMap, allDeps map[string]TargetNameMap,
	errs *errors.AggregatedError) {
	if unresolved[t.Name] != nil {
		delete(unresolved, t.Name)
		directDeps := make([]*Target, 0, len(allDeps[t.Name]))
		for _, dep := range allDeps[t.Name] {
			directDeps = append(directDeps, dep)
			m.resolveDeps(dep, unresolved, allDeps, errs)
		}
		for _, dep := range directDeps {
			for _, indirect := range allDeps[dep.Name] {
				allDeps[t.Name][indirect.Name] = indirect
			}
		}
		for _, dep := range allDeps[t.Name] {
			if dep.Name == t.Name {
				errs.Add(t.Errorf("cyclic dependency %s(%s)",
					dep.Name, dep.Source))
			}
		}
	}
}

// Merge merges settings s1 into s
func (s Settings) Merge(s1 Settings) error {
	if s1 == nil {
		return nil
	}
	return mapper.Map(s, s1)
}

// MergeFlat merges settings from a flat key/value map
// The key in flat map can be splitted by "." for a more complicated hierarchy
func (s Settings) MergeFlat(flat map[string]interface{}) error {
	for key, val := range flat {
		valDict, isValDict := val.(map[string]interface{})
		paths := strings.Split(key, ".")
		dict := s
		for n, path := range paths {
			sub, ok := dict[path].(map[string]interface{})
			if n+1 == len(paths) {
				if isValDict && ok {
					if err := mapper.Map(sub, valDict); err != nil {
						return err
					}
				} else {
					dict[path] = val
				}
			} else {
				if !ok {
					sub = make(Settings)
					dict[path] = sub
				}
				dict = sub
			}
		}
	}
	return nil
}

// IsEmpty indicates the watch list is empty
func (w WatchList) IsEmpty() bool {
	return len(w) == 0
}

// String formats the watch list as a string
func (w WatchList) String() string {
	if w.IsEmpty() {
		return ""
	}
	str := ""
	for _, item := range w {
		str += fmt.Sprintf("%s %d\n", item.Path, item.ModTime.Unix())
	}
	return str
}

// Digest calculates the digest based watched items
func (w WatchList) Digest() string {
	if w.IsEmpty() {
		return ""
	}
	h := sha1.Sum([]byte(w.String()))
	return hex.EncodeToString(h[0:])
}
