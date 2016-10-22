package main

import (
	"os"

	srv "github.com/evo-cloud/hmake/server"

	_ "github.com/evo-cloud/hmake/scm/github"
)

const (
	// Release is the major release version
	Release = "1.2.0"
)

// VersionSuffix is pre-release info
var VersionSuffix = "dev"

type serverCmd struct {
	ConfigFile  string `n:"config-file"`
	BindAddress string `n:"bind-address"`
	Port        int
	BaseURL     string `n:"base-url"`
}

func (c *serverCmd) Execute(args []string) error {
	cm := &LocalConfigManager{}
	err := cm.LoadFile(c.ConfigFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	overrides := map[string]interface{}{
		"port": c.Port,
	}
	if c.BindAddress != "" {
		overrides["bind-address"] = c.BindAddress
	}
	if c.BaseURL != "" {
		overrides["base-url"] = c.BaseURL
	}
	cm.Merge(overrides)
	s, err := srv.NewServer(cm)
	if err != nil {
		return err
	}
	return s.Serve()
}

func main() {
	cli(&serverCmd{}).Parse().Exec()
}
