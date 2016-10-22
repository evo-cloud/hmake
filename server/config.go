package server

import (
	"fmt"
	"io"
	"os"

	"github.com/easeway/langx.go/mapper"
	"github.com/pelletier/go-toml"
)

// Config is global configuration
type Config struct {
	BaseURL     string `json:"base-url"`
	BindAddress string `json:"bind-address"`
	Port        int    `json:"port"`
	TLSCertFile string `json:"tls-cert-file"`
	TLSKeyFile  string `json:"tls-key-file"`
}

// MissingConfig creates an error indicating configuration is missing
func MissingConfig(name string) error {
	return fmt.Errorf("missing configuration %s", name)
}

// ConfigManager manages configuration
type ConfigManager interface {
	Get(section string, out interface{}) error
}

// LocalConfigManager implements ConfigManager
// using simple configuration file
type LocalConfigManager struct {
	Raw map[string]interface{}
}

// Get implements ConfigManager
func (m *LocalConfigManager) Get(section string, out interface{}) error {
	if m.Raw != nil {
		return mapper.Map(out, m.Raw[section])
	}
	return nil
}

// Merge merges configuration
func (m *LocalConfigManager) Merge(input map[string]interface{}) error {
	if m.Raw == nil {
		m.Raw = make(map[string]interface{})
	}
	return mapper.Map(m.Raw, input)
}

// Load loads configuration from reader
func (m *LocalConfigManager) Load(reader io.Reader) error {
	tree, err := toml.LoadReader(reader)
	if err != nil {
		return err
	}
	return mapper.Map(m.Raw, tree.ToMap())
}

// LoadFile loads configuration from file
func (m *LocalConfigManager) LoadFile(fn string) error {
	f, err := os.Open(fn)
	if err == nil {
		err = m.Load(f)
		f.Close()
	}
	return err
}
