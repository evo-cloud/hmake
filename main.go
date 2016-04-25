package main

//go:generate cligen gen -f hmake-cli.yml -o hmake-cli.gen.go

import (
	"fmt"
	"strings"

	"github.com/codingbrain/clix.go/exts/bind"
	"github.com/codingbrain/clix.go/exts/help"
	"github.com/codingbrain/clix.go/term"

	"github.com/evo-cloud/hmake/docker"
	"github.com/evo-cloud/hmake/make"
)

const (
	faceOK  = ":-)"
	faceErr = ":-("
	faceNA  = ":-|"
)

type makeCmd struct {
	DefaultTargets []string `json:"default-targets"`
}

func (c *makeCmd) Execute(args []string) error {
	project, err := make.LoadProject()
	if len(args) == 0 {
		project.GetSettings(c)
		args = c.DefaultTargets
		if len(args) == 0 {
			return fmt.Errorf("at least one target is required from below:\n%s",
				strings.Join(project.TargetNames(), "\n"))
		}
	}
	plan := project.Plan()
	err = plan.Require(args...)
	if err != nil {
		return err
	}
	runner := docker.NewRunner(project)
	err = plan.Execute(func(t *make.Target) (result make.TaskResult, err error) {
		p := term.NewPrinter(term.Std)
		p.Styles("lightcyan").Print(":-> ").Pop().
			Styles(term.StyleB, term.StyleHi).Print(t.Name).Reset().Println()
		var style, face string
		result, err = runner(t)
		switch result {
		case make.Success:
			style = term.StyleOK
			face = faceOK
		case make.Failure:
			style = term.StyleErr
			face = faceErr
		case make.Skipped:
			style = term.StyleLo
			face = faceNA
		}
		p.Styles(style).Print(face+" ").Pop().
			Styles(term.StyleB, term.StyleHi).Print(t.Name).Pop()
		if err != nil {
			p.Print(" ").Styles(term.StyleErr).Print(err.Error()).Pop()
		}
		p.Reset().Println()
		return
	})
	if err != nil {
		return err
	}
	term.OK()
	return nil
}

type targetsCmd struct {
}

func pad(str string, maxLen int) string {
	for i := len(str); i < maxLen; i++ {
		str += " "
	}
	return str
}

func (c *targetsCmd) Execute([]string) error {
	project, err := make.LoadProject()
	if err != nil {
		return err
	}
	names := project.TargetNames()
	maxLen := -1
	for _, name := range names {
		if l := len(name); l > maxLen {
			maxLen = l
		}
	}
	for _, name := range project.TargetNames() {
		fmt.Println(pad(name, maxLen), project.Targets[name].Desc)
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
