// THIS FILE IS AUTO-GENERATED, DO NOT EDIT
package main

import "github.com/codingbrain/clix.go/args"

func cliDef() *args.CliDef {
    d := &args.CliDef{
        Cli: &args.Command{
            Name: "hmake",
            Desc: "HyperMake builds your project using consistent environment",
            Commands: []*args.Command{
                &args.Command{
                    Name: "make",
                    Desc: "Make default/specified targets",
                },
                &args.Command{
                    Name: "targets",
                    Desc: "List available targets in current project",
                },
            },
        },
    }
    d.Normalize()
    return d
}

