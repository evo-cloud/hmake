package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/ttacon/emoji"

	"github.com/codingbrain/clix.go/exts/bind"
	"github.com/codingbrain/clix.go/exts/help"
	"github.com/codingbrain/clix.go/term"

	"github.com/evo-cloud/hmake/docker"
	hm "github.com/evo-cloud/hmake/project"
	_ "github.com/evo-cloud/hmake/shell"
)

const (
	timeFmt = "15:04:05.000"
)

const (
	faceGo int = iota
	faceOK
	faceErr
	faceNA
	faceGood
)

var (
	facesNormal = []string{"=>", ":)", ":(", ":]", "OK"}
	facesEmoji  = []string{
		emoji.Emoji(":zap:"),
		emoji.Emoji(":yum:"),
		emoji.Emoji(":disappointed:"),
		emoji.Emoji(":expressionless:"),
		emoji.Emoji(":sunglasses:"),
	}
	faces = facesNormal
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
	Parallel   int
	RebuildAll bool `n:"rebuild-all"`
	Rebuild    []string
	JSON       bool
	Verbose    bool
	Color      bool
	Debug      bool
	Emoji      bool

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

	if c.Emoji {
		faces = facesEmoji
	}

	plan := p.Plan().OnEvent(c.onEvent)
	plan.Rebuild(c.Rebuild...)
	plan.RebuildAll = c.RebuildAll
	plan.MaxConcurrency = c.Parallel
	plan.DebugLog = c.Debug

	if err = plan.Require(args...); err != nil {
		return err
	}

	if err = plan.Execute(); err != nil {
		return err
	}
	term.NewPrinter(term.Std).Styles(term.StyleOK).Println(faces[faceGood])
	return nil
}

func (c *makeCmd) onEvent(event interface{}) {
	switch e := event.(type) {
	case *hm.EvtTaskStart:
		c.dumpEvent("start", e.Task)
		c.printTaskState(e.Task, faceGo, "lightblue",
			e.Task.StartTime.Format(timeFmt))
	case *hm.EvtTaskFinish:
		c.dumpEvent("finish", e.Task)
		extra := e.Task.FinishTime.Format(timeFmt) +
			" [+" + e.Task.Duration().String() + "]"
		switch e.Task.Result {
		case hm.Success:
			c.printTaskState(e.Task, faceOK, term.StyleOK, extra)
		case hm.Failure:
			c.printTaskState(e.Task, faceErr, term.StyleErr, extra)
			if !c.Verbose {
				c.printFailedTaskOutput(e.Task)
			}
		case hm.Skipped:
			c.printTaskState(e.Task, faceNA, term.StyleLo, "")
		}
	case *hm.EvtTaskOutput:
		c.dumpTaskOutput(e.Task, e.Output)
		if c.Verbose {
			c.printTaskOutput(e.Task, e.Output)
		}
	}
}

func (c *makeCmd) printTaskState(task *hm.Task, face int, style, extra string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	out := term.NewPrinter(term.Std)
	if c.noNewLine != "" {
		c.noNewLine = ""
		out.Println()
	}
	out.Styles(term.StyleB, style).Print(faces[face]+" ").Pop().
		Styles(term.StyleB, term.StyleHi).Print(task.Name()).Pop()
	if extra != "" {
		out.Styles(term.StyleLo).Print(" " + extra).Pop()
	}
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

func (c *makeCmd) dumpEvent(event string, task *hm.Task) {
	if !c.JSON {
		return
	}
	e := map[string]interface{}{
		"event":  event,
		"target": task.Name(),
		"state":  task.State.String(),
	}
	if task.State >= hm.Running {
		e["start-at"] = task.StartTime
	}
	if task.State >= hm.Finished {
		e["result"] = task.Result.String()
		e["finish-at"] = task.FinishTime
	}

	encoded, err := json.Marshal(e)
	if err != nil {
		return
	}
	c.lock.Lock()
	fmt.Println(string(encoded))
	c.lock.Unlock()
}

func (c *makeCmd) dumpTaskOutput(task *hm.Task, out []byte) {
	if !c.JSON {
		return
	}
	e := map[string]interface{}{
		"event":    "output",
		"target":   task.Name(),
		"state":    task.State.String(),
		"start-at": task.StartTime,
		"output":   out,
	}
	encoded, err := json.Marshal(e)
	if err != nil {
		return
	}
	c.lock.Lock()
	fmt.Println(string(encoded))
	c.lock.Unlock()
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
