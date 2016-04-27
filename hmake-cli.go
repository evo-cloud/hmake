package main

import "github.com/codingbrain/clix.go/args"

func cliDef() *args.CliDef {
	d := &args.CliDef{
		Cli: &args.Command{
			Name: "hmake",
			Desc: "HyperMake builds your project using consistent environment",
			Options: []*args.Option{
				&args.Option{
					Name:  "verbose",
					Alias: []string{"v"},
					Desc:  "Display more information",
					Type:  "bool",
				},
				&args.Option{
					Name:    "color",
					Desc:    "Colored output",
					Type:    "bool",
					Default: true,
				},
			},
		},
	}
	d.Normalize()
	return d
}
