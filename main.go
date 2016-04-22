package main

//go:generate cligen gen -f hmake-cli.yml -o hmake-cli.gen.go

import (
	"fmt"
	"sort"

	"github.com/codingbrain/clix.go/exts/bind"
	"github.com/codingbrain/clix.go/exts/help"
	"github.com/codingbrain/clix.go/term"

	"github.com/evo-cloud/hmake/make"
)

type makeCmd struct {
}

func (c *makeCmd) Execute([]string) error {
	term.OK()
	return nil
}

type targetsCmd struct {
}

func (c *targetsCmd) Execute([]string) error {
	project := &make.Project{}
	err := project.Locate()
	if err != nil {
		return fmt.Errorf("project not found: %v", err)
	}
	err = project.Scan()
	if err != nil {
		return fmt.Errorf("project scan failed: %v", err)
	}
	var targets []string
	for _, s := range project.Schemas {
		for name := range s.Targets {
			targets = append(targets, name)
		}
	}
	sort.Strings(targets)
	for _, name := range targets {
		fmt.Println(name)
	}
	term.OK()
	return nil
}

func main() {
	cliDef().
		Use(
			bind.NewExt().
				Bind(&makeCmd{}, "make").
				Bind(&targetsCmd{}, "targets")).
		Use(help.NewExt()).
		Parse().
		Exec()
}
