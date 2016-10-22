package main

import (
	"github.com/codingbrain/clix.go/exts/bind"
	"github.com/codingbrain/clix.go/exts/help"
	"github.com/codingbrain/clix.go/flag"
	"github.com/codingbrain/clix.go/term"
)

func cli(cmd *serverCmd) *flag.CliDef {
	cliDef := &flag.CliDef{
		Cli: &flag.Command{
			Name: "hmaked",
			Desc: "HyperMake Server",
			Options: []*flag.Option{
				&flag.Option{
					Name:    "config-file",
					Alias:   []string{"c"},
					Desc:    "Specify configuration file, ignored if fail to load",
					Default: "/etc/hmaked.conf",
				},
				&flag.Option{
					Name: "bind-address",
					Desc: "Address of interface to listen on",
				},
				&flag.Option{
					Name:    "port",
					Desc:    "Listening port",
					Type:    "integer",
					Default: 1180,
				},
				&flag.Option{
					Name: "base-url",
					Desc: "Base URL for external access, like http://hostname",
				},
			},
		},
	}
	cliDef.Normalize()
	cliDef.Use(term.NewExt()).
		Use(bind.NewExt().Bind(cmd)).
		Use(help.NewExt())
	return cliDef
}
