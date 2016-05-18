package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	cv "github.com/easeway/go-cliview"
	"github.com/easeway/langx.go/errors"
	"github.com/ttacon/emoji"

	"github.com/codingbrain/clix.go/exts/bind"
	"github.com/codingbrain/clix.go/exts/help"
	"github.com/codingbrain/clix.go/term"

	"github.com/evo-cloud/hmake/docker"
	hm "github.com/evo-cloud/hmake/project"
	_ "github.com/evo-cloud/hmake/shell"
)

const (
	// Release is the major release version
	Release = "1.0.0"
	// Version is full version string
	Version = Release + VersionSuffix
	// Website is the URL to project website
	Website = "https://github.com/evo-cloud/hmake"

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
	Chdir       string
	Include     []string
	Define      map[string]interface{}
	Parallel    int
	RebuildAll  bool `n:"rebuild-all"`
	Rebuild     []string
	Skip        []string
	RcFile      bool
	JSON        bool
	Summary     bool
	Verbose     bool
	Banner      bool
	Color       bool
	Emoji       bool
	Debug       bool
	ShowSummary bool `n:"show-summary"`
	ShowTargets bool `n:"targets"`
	DryRun      bool
	Version     bool

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

func (c *makeCmd) Execute(args []string) (err error) {
	if c.Version {
		if c.JSON {
			encoded, _ := json.Marshal(c.Version)
			fmt.Println(string(encoded))
		} else {
			term.Println(Version)
		}
		return
	}

	if c.Chdir != "" {
		if err = os.Chdir(c.Chdir); err != nil {
			return
		}
	}

	if c.Banner {
		c.showBanner()
	}

	var p *hm.Project
	if p, err = hm.LocateProject(); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Unable to find HyperMake")
		}
		return
	}

	if c.ShowSummary {
		err = c.showSummary(p, nil)
		return
	}

	if err = p.Resolve(); err != nil {
		return
	}

	incErrs := &errors.AggregatedError{}
	if c.RcFile {
		incErrs.Add(p.LoadRcFiles())
	}
	for _, inc := range c.Include {
		_, e := p.Load(inc)
		incErrs.Add(e)
	}
	if err = incErrs.Aggregate(); err != nil {
		return
	}
	if err = p.Finalize(); err != nil {
		return
	}

	if c.Define != nil {
		if err = p.MergeSettingsFlat(c.Define); err != nil {
			return
		}
	}

	names := p.TargetNames()
	padLen := 0
	for _, name := range names {
		if l := len(name); l > padLen {
			padLen = l
		}
	}
	if c.ShowTargets {
		c.showTargets(p, names, padLen)
		return
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
			c.showTargets(p, names, padLen)
			return fmt.Errorf("No targets selected, please choose at least one from above")
		}
	}

	if c.Emoji {
		faces = facesEmoji
	}

	plan := p.Plan()
	plan.Env["HMAKE_VERSION"] = Version
	plan.OnEvent(c.onEvent)
	errs := &errors.AggregatedError{}
	plan.Rebuild(p.Targets.CompleteNames(c.Rebuild, errs)...)
	plan.Skip(p.Targets.CompleteNames(c.Skip, errs)...)
	requires := p.Targets.CompleteNames(args, errs)
	if err = errs.Aggregate(); err != nil {
		return
	}
	plan.RebuildAll = c.RebuildAll
	plan.MaxConcurrency = c.Parallel
	plan.DebugLog = c.Debug
	plan.DryRun = c.DryRun

	if err = plan.Require(requires...); err != nil {
		return
	}

	err = plan.Execute()
	if c.Summary {
		c.showSummary(p, plan)
	}

	if err != nil {
		return
	}

	term.NewPrinter(term.Std).Styles(term.StyleOK).Println(faces[faceGood])
	return
}

func (c *makeCmd) showBanner() {
	out := term.NewPrinter(term.Std)
	out.Styles("lightyellow", term.StyleB).Print("HyperMake").Pop().
		Styles(term.StyleHi).Print(" v"+Version+" ").Pop().
		Styles("lightblue", "underline").Println(Website).Pop().
		Println()
}

func (c *makeCmd) showTargets(p *hm.Project, names []string, padLen int) {
	if c.JSON {
		data := make([]map[string]string, 0, len(p.Targets))
		for _, name := range names {
			t := p.Targets[name]
			data = append(data, map[string]string{
				"name":        t.Name,
				"description": t.Desc,
			})
		}
		encoded, _ := json.Marshal(data)
		fmt.Println(string(encoded))
	} else {
		settings := &projectSettings{}
		p.GetSettings(settings)

		out := term.NewPrinter(term.Std)
		for _, name := range names {
			t := p.Targets[name]
			found := false
			for _, n := range settings.DefaultTargets {
				if n == name {
					found = true
					break
				}
			}
			if found {
				out.Styles(term.StyleOK, term.StyleB).Print(" * ").Pop()
			} else {
				out.Print("   ")
			}
			out.Styles(term.StyleHi, term.StyleB).
				Print(pad(name, padLen+2)).Pop().Println(t.Desc)
		}
	}
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

func stylerPrint(text string, styles ...string) string {
	var buf bytes.Buffer
	term.NewPrinter(&buf).Styles(styles...).Print(text)
	return buf.String()
}

func headStyler(class, text string, data interface{}) string {
	if strings.HasPrefix(class, "table:head:") {
		return stylerPrint(text, term.StyleHi, term.StyleB)
	}
	return text
}

func resultStyler(class, text string, data interface{}) string {
	if strings.HasPrefix(class, "table:row:") {
		switch text {
		case hm.Success.String():
			return stylerPrint(text, term.StyleOK)
		case hm.Failure.String():
			return stylerPrint(text, term.StyleErr)
		case hm.Skipped.String():
			return stylerPrint(text, term.StyleLo)
		default:
			return text
		}
	}
	return headStyler(class, text, data)
}

func errorStyler(class, text string, data interface{}) string {
	if strings.HasPrefix(class, "table:row:") {
		return stylerPrint(text, term.StyleErr)
	}
	return headStyler(class, text, data)
}

func timeFetcher(col cv.Column, row map[string]interface{}) interface{} {
	if strVal, ok := row[col.Field].(string); ok && strVal != "" {
		var t time.Time
		if t.UnmarshalText([]byte(strVal)) == nil {
			return t.Format(timeFmt)
		}
	}
	return ""
}

func durationFetcher(col cv.Column, row map[string]interface{}) interface{} {
	var startAtStr, finishAtStr string
	var startAt, finishAt time.Time
	var ok bool
	if startAtStr, ok = row["start-at"].(string); !ok || startAtStr == "" {
		return ""
	}
	if finishAtStr, ok = row["finish-at"].(string); !ok || finishAtStr == "" {
		return ""
	}
	if err := startAt.UnmarshalText([]byte(startAtStr)); err != nil {
		return ""
	}
	if err := finishAt.UnmarshalText([]byte(finishAtStr)); err != nil {
		return ""
	}
	duration := finishAt.Sub(startAt)
	if duration == 0 {
		return ""
	}
	return duration.String()
}

func (c *makeCmd) showSummary(p *hm.Project, plan *hm.ExecPlan) error {
	var sum []map[string]interface{}
	if plan != nil {
		sum = plan.Summary
	} else {
		data, err := ioutil.ReadFile(p.SummaryFile())
		if err != nil {
			return err
		}

		err = json.Unmarshal(data, &sum)
		if err != nil {
			return err
		}
	}

	if c.JSON {
		encoded, _ := json.Marshal(sum)
		fmt.Println(string(encoded))
	}

	table := &cv.Table{
		Output: cv.Output{
			Writer: term.Std,
			Styler: headStyler,
		},
		Border: cv.BorderCompact,
		Columns: []cv.Column{
			{Title: "Target", Field: "target"},
			{Title: "Result", Field: "result", Styler: resultStyler},
			{Title: "Duration", Align: cv.AlignRight, Fetcher: durationFetcher},
			{Title: "Start", Field: "start-at", Fetcher: timeFetcher},
			{Title: "Finish", Field: "finish-at", Fetcher: timeFetcher},
			{Title: "Error", Field: "error", Styler: errorStyler},
		},
	}

	w, _, e := term.Size()
	if e != nil || w <= 0 {
		w = 80
	}
	table.MaxWidth = w
	table.Print(sum)
	return nil
}

func init() {
	hm.DefaultExecDriver = docker.ExecDriverName
}

func main() {
	cliDef().
		Use(term.NewExt()).
		Use(bind.NewExt().Bind(&makeCmd{})).
		Use(help.NewExt()).
		Parse().
		Exec()
}
