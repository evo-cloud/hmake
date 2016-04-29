package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/codingbrain/clix.go/exts/bind"
	"github.com/codingbrain/clix.go/exts/help"
	"github.com/codingbrain/clix.go/term"

	"github.com/evo-cloud/hmake/docker"
	hm "github.com/evo-cloud/hmake/project"
	_ "github.com/evo-cloud/hmake/shell"
)

const (
	faceGo  = "=>"
	faceOK  = ":)"
	faceErr = ":("
	faceNA  = ":]"
)

type taskState struct {
	color  string
	prefix string
}

type projectSettings struct {
	DefaultTargets []string `json:"default-targets"`
}

type makeCmd struct {
	// command line options
	Parallel int
	Rebuild  bool
	Verbose  bool
	Color    bool

	settings  projectSettings
	tasks     map[string]*taskState
	noNewLine string // name of task printed the last output
	lock      sync.Mutex
}

var colors = []string{
	"blue", "yellow", "green", "magenta", "cyan", "darkgray",
	"lightblue", "lightyellow", "lightgreen",
	"lightmagenta", "lightcyan", "lightgray",
}

func pad(str string, l int) string {
	for i := len(str); i < l; i++ {
		str += " "
	}
	return str
}

func (c *makeCmd) Execute(args []string) error {
	p, err := hm.LoadProject()
	if err != nil {
		return err
	}

	names := p.TargetNames()
	padLen := 0
	for _, name := range names {
		if l := len(name); l > padLen {
			padLen = l
		}
	}
	c.tasks = make(map[string]*taskState)
	for n, name := range names {
		c.tasks[name] = &taskState{
			color:  colors[n%len(colors)],
			prefix: "[" + pad(name+"]", padLen+1) + " ",
		}
	}

	if len(args) == 0 {
		p.GetSettings(&c.settings)
		args = c.settings.DefaultTargets
		if len(args) == 0 {
			return fmt.Errorf("at least one target is required from below:\n%s",
				strings.Join(names, "\n"))
		}
	}

	plan := p.Plan().OnEvent(c.onEvent)
	plan.RebuildAll = c.Rebuild
	plan.MaxConcurrency = c.Parallel

	if err = plan.Require(args...); err != nil {
		return err
	}

	if err = plan.Execute(); err != nil {
		return err
	}
	term.OK()
	return nil
}

func (c *makeCmd) onEvent(event interface{}) {
	switch e := event.(type) {
	case *hm.EvtTaskStart:
		c.printTaskState(e.Task, faceGo, "lightblue")
	case *hm.EvtTaskFinish:
		switch e.Task.Result {
		case hm.Success:
			c.printTaskState(e.Task, faceOK, term.StyleOK)
		case hm.Failure:
			c.printTaskState(e.Task, faceErr, term.StyleErr)
			if !c.Verbose {
				c.printFailedTaskOutput(e.Task)
			}
		case hm.Skipped:
			c.printTaskState(e.Task, faceNA, term.StyleLo)
		}
	case *hm.EvtTaskOutput:
		if c.Verbose {
			c.printTaskOutput(e.Task, e.Output)
		}
	}
}

func (c *makeCmd) printTaskState(task *hm.Task, face, style string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	out := term.NewPrinter(term.Std)
	if c.noNewLine != "" {
		c.noNewLine = ""
		out.Println()
	}
	out.Styles(term.StyleB, style).Print(face+" ").Pop().
		Styles(term.StyleB, term.StyleHi).Print(task.Name()).Pop()
	if task.Error != nil {
		out.Print(" ").Styles(term.StyleErr).Print(task.Error.Error()).Pop()
	}
	out.Println()
}

func (c *makeCmd) printTaskOutput(task *hm.Task, p []byte) {
	c.lock.Lock()
	defer c.lock.Unlock()

	s := c.tasks[task.Name()]
	out := term.NewPrinter(term.Std).Styles(s.color)
	lines := strings.Split(strings.Replace(string(p), "\r\n", "\n", -1), "\n")
	for n, line := range lines {
		continueLine := n == 0 && c.noNewLine == task.Name()

		if n == len(lines)-1 {
			if line != "" {
				c.noNewLine = task.Name()
			} else {
				c.noNewLine = ""
				break
			}
		}

		if !continueLine {
			out.Print(s.prefix)
		}

		rets := strings.Split(line, "\r")
		maxLen := 0
		for _, ret := range rets {
			if l := len(ret); l > maxLen {
				maxLen = l
				line = ret
			}
		}

		if continueLine && len(rets) > 1 {
			out.Print("\r" + s.prefix)
		}

		out.Write([]byte(line))
		if n < len(lines)-1 {
			out.Println()
		}
	}
}

func (c *makeCmd) printFailedTaskOutput(task *hm.Task) {
	f, err := os.Open(task.LogFile())
	c.lock.Lock()
	defer c.lock.Unlock()
	if err != nil {
		term.Errorln("Dump output failed: " + err.Error())
		return
	}
	defer f.Close()
	out := term.NewPrinter(term.Std).Styles(term.StyleErr)
	io.Copy(out, f)
	out.Println()
}

func init() {
	hm.DefaultExecDriver = docker.ExecDriverName
}

func main() {
	cliDef().
		Use(bind.NewExt().Bind(&makeCmd{})).
		Use(help.NewExt()).
		Parse().
		Exec()
}
