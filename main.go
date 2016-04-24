package main

//go:generate cligen gen -f hmake-cli.yml -o hmake-cli.gen.go

import (
	"fmt"
	"strings"

	"github.com/codingbrain/clix.go/exts/bind"
	"github.com/codingbrain/clix.go/exts/help"
	"github.com/codingbrain/clix.go/term"

	"github.com/evo-cloud/hmake/make"
)

type makeCmd struct {
}

func (c *makeCmd) Execute(args []string) error {
	project, err := make.LoadProject()
	if len(args) == 0 {
		return fmt.Errorf("at least one target is required from below:\n%s",
			strings.Join(project.TargetNames(), "\n"))
	}
	plan := project.Plan()
	err = plan.Require(args...)
	if err != nil {
		return err
	}
	err = plan.Execute(func(t *make.Target) (make.TaskResult, error) {
		fmt.Println("Exec", t.Name)
		return make.Success, nil
	})
	if err != nil {
		return err
	}
	term.OK()
	return nil
}

type targetsCmd struct {
}

func (c *targetsCmd) Execute([]string) error {
	project, err := make.LoadProject()
	if err != nil {
		return err
	}
	for _, name := range project.TargetNames() {
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
