package make

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/codingbrain/clix.go/clix"
	"github.com/codingbrain/gms"
	"github.com/codingbrain/mapper.go/mapper"
	yaml "gopkg.in/yaml.v2"
)

const (
	// Format is the supported format
	Format = "hypermake.v0"
	// RootFile is hmake filename sits on root
	RootFile = "HyperMake"
	// Suffix is suffix of alternative hmake file name
	Suffix = ".hmake"
	// RcFile is file name only for settings
	RcFile = "hmakerc"
)

// ErrUnsupportedFormat indicates the file is not recognized
var ErrUnsupportedFormat = errors.New("unsupported format")

// Target defines a build target
type Target struct {
	Name   string                 `json:"name"`
	Before []string               `json:"before"`
	After  []string               `json:"after"`
	Envs   []string               `json:"envs"`
	Cmds   []*Command             `json:"cmds"`
	Ext    map[string]interface{} `json:"*"`

	// Source is the file defined the target
	Source string `json:"-"`
}

// Command defines a single command to execute
type Command struct {
	Shell string                 `json:"*"`
	Ext   map[string]interface{} `json:"*"`
}

// Settings applies to targets
type Settings struct {
	Properties map[string]interface{} `json:"*"`

	Source string `json:"-"`
	Scope  string `json:"-"`
}

// Schema defines the content in a file
type Schema struct {
	Format   string                 `json:"format"`
	Targets  map[string]*Target     `json:"targets"`
	Settings map[string]interface{} `json:"settings"`

	Source string `json:"-"`
}

// Project is the world view of hmake
type Project struct {
	BaseDir  string
	Schemas  []*Schema
	Settings []*Settings
}

func loadYaml(filename string) (map[string]interface{}, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	val := make(map[string]interface{})
	return val, yaml.Unmarshal(data, val)
}

// Locate looks up the root directory of project where HyperMake exists
func (p *Project) Locate() error {
	if p.BaseDir != "" {
		return nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	for {
		data, err := loadYaml(filepath.Join(wd, RootFile))
		if err == nil {
			format, ok := data["format"].(string)
			if ok && format == Format {
				p.BaseDir = wd
				return nil
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		dir := filepath.Dir(wd)
		if dir == wd {
			break
		}
		wd = dir
	}

	return os.ErrNotExist
}

// Load a single hmake file or hmakerc
func (p *Project) Load(filename string, asRc bool) error {
	val, err := loadYaml(filepath.Join(p.BaseDir, filename))
	if err != nil {
		return err
	}

	m := &mapper.Mapper{}
	if asRc {
		settings := &Settings{}
		err = m.Map(settings, val)
		if err == nil {
			settings.Source = filename
			settings.Scope = filepath.Dir(filename)
			p.Settings = append(p.Settings, settings)
		}
	} else {
		if format, ok := val["format"].(string); !ok || format != Format {
			return fmt.Errorf("unsupported format: " + format)
		}
		schema := &Schema{}
		err = m.Map(schema, val)
		if err == nil {
			schema.Source = filename
			p.Schemas = append(p.Schemas, schema)
		}
	}
	return err
}

// Scan the whole project and load all available files
func (p *Project) Scan() error {
	errs := clix.AggregatedError{}
	// always load root file
	errs.Add(p.Load(RootFile, false))
	// populate root settings
	if len(p.Schemas) > 0 && p.Schemas[0].Settings != nil {
		p.Settings = append(p.Settings, &Settings{
			Properties: p.Schemas[0].Settings,
			Source:     RootFile,
			Scope:      "",
		})
	}

	// scan project
	walker := &gms.RepoWalker{BreadthFirst: true, PathPrefix: p.BaseDir + "/"}
	walker.Use(func(item *gms.WalkingItem) (bool, error) {
		if item.FileInfo.IsDir() {
			return !strings.HasPrefix(item.Name, ".") &&
				!p.IsIgnored(filepath.Join(item.Path, item.Name)), nil
		}
		return item.Name == RcFile || strings.HasSuffix(item.Name, Suffix), nil
	})
	walker.WalkerFn = func(item gms.WalkingItem) error {
		if item.FileInfo.IsDir() {
			return nil
		}
		err := p.Load(filepath.Join(item.Path, item.Name), item.Name == RcFile)
		errs.Add(err)
		// always returns nil as error is aggregated
		return nil
	}
	errs.Add(walker.Visit("", p))

	return errs.Aggregate()
}

// IsIgnored determines whether the path should be ignored
func (p *Project) IsIgnored(path string) bool {
	// TODO
	return false
}

// BasePath implements gms.Repository
func (p *Project) BasePath() string {
	return ""
}

// Persist implements gms.Repository
func (p *Project) Persist() (h gms.PersistentHandle) {
	return
}
