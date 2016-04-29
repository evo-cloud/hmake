package main

import "github.com/codingbrain/clix.go/args"

func cliDef() *args.CliDef {
	d := &args.CliDef{
		Cli: &args.Command{
			Name: "hmake",
			Desc: "HyperMake builds your project using consistent environment",
			Options: []*args.Option{
				&args.Option{
					Name:    "parallel",
					Alias:   []string{"p"},
					Desc:    "Set maximum number of targets executed in parallel, 0 for auto, -1 for unlimited",
					Type:    "int",
					Default: 0,
				},
				&args.Option{
					Name:  "rebuild",
					Alias: []string{"r"},
					Desc:  "Rebuild everything regardless of cached state",
					Type:  "bool",
				},
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
