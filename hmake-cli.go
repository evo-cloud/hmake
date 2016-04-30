package main

import (
	"os"
	"runtime"
	"strings"

	"github.com/codingbrain/clix.go/args"
)

func emojiSupported() bool {
	return runtime.GOOS == "darwin" &&
		strings.HasSuffix(strings.ToLower(os.Getenv("LANG")), "utf-8")
}

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
					Name:  "rebuild-all",
					Alias: []string{"R"},
					Desc:  "Rebuild everything regardless of cached state",
					Type:  "bool",
				},
				&args.Option{
					Name:  "rebuild",
					Alias: []string{"r"},
					Desc:  "Rebuild specified targets",
					List:  true,
				},
				&args.Option{
					Name: "json",
					Desc: "Dump events in JSON to stdout",
					Type: "bool",
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
				&args.Option{
					Name:    "emoji",
					Desc:    "Output emoji",
					Type:    "bool",
					Default: emojiSupported(),
				},
				&args.Option{
					Name: "debug",
					Desc: "Write debug log into .hmake/hmake.debug.log",
					Type: "bool",
				},
			},
		},
	}
	d.Normalize()
	return d
}
