package make

import (
	"fmt"

	"github.com/easeway/langx.go/errors"
	"github.com/easeway/langx.go/mapper"
)

// Target defines a build target
type Target struct {
	Name   string                 `json:"name"`
	Desc   string                 `json:"description"`
	Before []string               `json:"before"`
	After  []string               `json:"after"`
	Envs   []string               `json:"envs"`
	Cmds   []*Command             `json:"cmds"`
	Script string                 `json:"script"`
	Ext    map[string]interface{} `json:"*"`

	// Source is the file defined the target
	Source string `json:"-"`

	// Settings is the stack of settings
	Settings []Settings
	// Depends is the dependencies
	Depends TargetNameMap
	// Activates is the opposite of Depends
	Activates TargetNameMap
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

// Initialize prepare fields in target
func (t *Target) Initialize(name string, settings []Settings) {
	t.Name = name
	t.Settings = settings
	t.Depends = make(TargetNameMap)
	t.Activates = make(TargetNameMap)
}

// GetExt maps Ext to provided value
func (t *Target) GetExt(v interface{}) error {
	if t.Ext != nil {
		m := &mapper.Mapper{}
		return m.Map(v, t.Ext)
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
func (s Settings) Merge(s1 Settings) {
	if s1 == nil {
		return
	}

	for k, val := range s1 {
		if s.mergeList(k, val) {
			continue
		}
		s[k] = val
	}
}

func (s Settings) mergeList(key string, val interface{}) bool {
	dest, exist := s[key]
	if !exist {
		return false
	}
	vList, ok := val.([]interface{})
	if !ok {
		return false
	}
	dList, ok := dest.([]interface{})
	if !ok {
		return false
	}
	if len(vList) == 0 {
		return true
	}
	if str, ok := vList[0].(string); ok && str == "$new" {
		s[key] = vList[1:]
		return true
	}
	for _, v := range vList {
		dList = append(dList, v)
		s[key] = dList
	}
	return true
}
