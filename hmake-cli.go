package main

import (
	"strings"

	"github.com/codingbrain/clix.go/exts/bind"
	"github.com/codingbrain/clix.go/exts/help"
	"github.com/codingbrain/clix.go/flag"
	"github.com/codingbrain/clix.go/term"
)

type execFilterExt struct {
}

func (x *execFilterExt) RegisterExt(parser *flag.Parser) {
	parser.AddParseExt(flag.EvtAssignOpt, x)
}

func (x *execFilterExt) HandleParseEvent(event string, ctx *flag.ParseContext) {
	if ctx.Option != nil &&
		strings.HasPrefix(ctx.Option.Name, "exec") {
		ctx.ParseEnd()
	}
}

func cliDef(cmd *makeCmd) *flag.CliDef {
	d := &flag.CliDef{
		Cli: &flag.Command{
			Name: "hmake",
			Desc: "HyperMake builds projects without pre-requisites\n" + Website,
			Options: []*flag.Option{
				&flag.Option{
					Name:  "chdir",
					Alias: []string{"C"},
					Desc:  "Change to specified directory before doing anything",
					Tags:  map[string]interface{}{"help-var": "PATH"},
				},
				&flag.Option{
					Name:  "file",
					Alias: []string{"f"},
					Desc:  "Specify the project filename instead of default HyperMake",
					Tags:  map[string]interface{}{"help-var": "FILE"},
					Type:  "string",
				},
				&flag.Option{
					Name:    "include",
					Alias:   []string{"I"},
					Desc:    "Include additional files inside project directory, relative path",
					Example: "-I custom.hmake -I custom1.hmake",
					List:    true,
					Tags:    map[string]interface{}{"help-var": "FILE"},
				},
				&flag.Option{
					Name:    "property",
					Alias:   []string{"P"},
					Desc:    "Define additional setting property",
					Example: "--property docker.src-volume=/tmp/src -P docker.image=myimage",
					Type:    "dict",
					Tags:    map[string]interface{}{"help-var": "KEY=VAL"},
				},
				&flag.Option{
					Name:    "parallel",
					Alias:   []string{"p"},
					Desc:    "Set maximum number of targets executed in parallel, 0 for auto, -1 for unlimited",
					Example: "--parallel=4",
					Type:    "int",
					Default: 0,
					Tags:    map[string]interface{}{"help-var": "NUM"},
				},
				&flag.Option{
					Name:  "rebuild-all",
					Alias: []string{"R"},
					Desc:  "Rebuild everything regardless of cached state",
					Type:  "bool",
				},
				&flag.Option{
					Name:  "rebuild-target",
					Alias: []string{"r"},
					Desc:  "Rebuild specified targets",
					List:  true,
					Tags:  map[string]interface{}{"help-var": "TARGET"},
				},
				&flag.Option{
					Name:  "rebuild",
					Alias: []string{"b"},
					Desc:  "Rebuild targets specified on command line",
					Type:  "bool",
				},
				&flag.Option{
					Name:  "skip",
					Alias: []string{"S"},
					Desc:  "Skip the execution of specified target",
					List:  true,
					Tags:  map[string]interface{}{"help-var": "TARGET"},
				},
				&flag.Option{
					Name:  "exec",
					Alias: []string{"x"},
					Desc: "Execute a shell command in the context of a target.\n" +
						"The target name must be specified in settings.exec-target " +
						"or use --exec-with=TARGET.\n" +
						"It's extremely useful to run arbitrary command in the context " +
						"of a target.\n" +
						"It should come as the last option, " +
						"as the rest command-line arguments will be treated as a shell command.",
					Example: "hmake -x go version\n" +
						"hmake -x   # enter an interactive shell\n",
					Type: "bool",
				},
				&flag.Option{
					Name:  "exec-with",
					Alias: []string{"X"},
					Desc: "Explicitly specify the target for --exec instead of fetching " +
						"from settings.exec-target.\n" +
						"As it implies --exec, it should come as the last option.",
					Example: "hmake --exec-with=vendor go version\n",
					Type:    "string",
				},
				&flag.Option{
					Name:    "rcfile",
					Desc:    "Load .hmakerc files inside project directories",
					Type:    "bool",
					Default: true,
				},
				&flag.Option{
					Name: "json",
					Desc: "Dump events in JSON to stdout",
					Type: "bool",
				},
				&flag.Option{
					Name:    "summary",
					Alias:   []string{"s"},
					Desc:    "Show summary when make completes",
					Type:    "bool",
					Default: true,
				},
				&flag.Option{
					Name:  "verbose",
					Alias: []string{"v"},
					Desc:  "Display more information",
					Type:  "bool",
				},
				&flag.Option{
					Name:  "quiet",
					Alias: []string{"q"},
					Desc:  "Display less information, suppress the output",
					Type:  "bool",
				},
				&flag.Option{
					Name:    "banner",
					Desc:    "Show banner",
					Type:    "bool",
					Default: true,
				},
				&flag.Option{
					Name:    "color",
					Desc:    "Colored output",
					Type:    "bool",
					Default: true,
				},
				&flag.Option{
					Name:    "emoji",
					Desc:    "Output emoji",
					Type:    "bool",
					Default: false,
				},
				&flag.Option{
					Name:    "debug-log",
					Desc:    "Write debug log into .hmake/hmake.debug.log",
					Type:    "bool",
					Default: true,
				},
				&flag.Option{
					Name: "show-summary",
					Desc: "Show summary of last build and exit",
					Type: "bool",
				},
				&flag.Option{
					Name: "targets",
					Desc: "Print targets and exit",
					Type: "bool",
				},
				&flag.Option{
					Name: "dryrun",
					Desc: "Show the execution of targets without doing anything",
					Type: "bool",
				},
				&flag.Option{
					Name: "version",
					Desc: "Display version and exit",
					Type: "bool",
				},
			},
		},
	}
	d.Normalize()
	return d.Use(term.NewExt()).
		Use(&execFilterExt{}).
		Use(bind.NewExt().Bind(cmd)).
		Use(help.NewExt())
}
