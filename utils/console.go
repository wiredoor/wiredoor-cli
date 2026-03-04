package utils

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/briandowns/spinner"
	"github.com/mattn/go-isatty"
)

type ProgressMode string

const (
	ProgressAuto   ProgressMode = "auto"
	ProgressAlways ProgressMode = "always"
	ProgressNever  ProgressMode = "never"
)

type ConsoleOptions struct {
	Out io.Writer // default os.Stdout
	Err io.Writer // default os.Stderr

	Quiet bool

	ProgressMode ProgressMode // auto|always|never
}

type Console struct {
	out io.Writer
	err io.Writer

	quiet bool

	mode ProgressMode

	mu   sync.Mutex
	sp   *spinner.Spinner
	spOn bool
}

var (
	once sync.Once
	c    *Console
)

func InitConsole(opts ConsoleOptions) {
	once.Do(func() {
		c = NewConsole(opts)
	})
}

func Terminal() *Console {
	if c == nil {
		c = NewConsole(ConsoleOptions{})
	}
	return c
}

func NewConsole(opts ConsoleOptions) *Console {
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	errW := opts.Err
	if errW == nil {
		errW = os.Stderr
	}
	mode := opts.ProgressMode
	if mode == "" {
		mode = ProgressAuto
	}

	return &Console{
		out:   out,
		err:   errW,
		quiet: opts.Quiet,
		mode:  mode,
	}
}

// ---------- Human output (stdout)

func (c *Console) Blank() {
	if c.quiet {
		return
	}
	fmt.Fprintln(c.out)
}

func (c *Console) Section(title string) {
	if c.quiet {
		return
	}
	fmt.Fprintln(c.out, title)
}

func (c *Console) Println(a ...any) {
	if c.quiet {
		return
	}
	fmt.Fprintln(c.out, a...)
}

func (c *Console) Printf(format string, args ...any) {
	if c.quiet {
		return
	}
	fmt.Fprintf(c.out, format+"\n", args...)
}

func (c *Console) KV(key string, value any) {
	if c.quiet {
		return
	}
	fmt.Fprintf(c.out, "%s: %v\n", key, value)
}

func (c *Console) Hint(msg string) {
	if c.quiet {
		return
	}
	fmt.Fprintf(c.out, "Hint: %s\n", msg)
}

func (c *Console) Table(headers []string, rows [][]string) {
	if c.quiet {
		return
	}
	w := tabwriter.NewWriter(c.out, 0, 0, 2, ' ', 0)
	if len(headers) > 0 {
		fmt.Fprintln(w, strings.Join(headers, "\t"))
	}
	for _, r := range rows {
		fmt.Fprintln(w, strings.Join(r, "\t"))
	}
	_ = w.Flush()
}

// ---------- Diagnostics (stderr)

func (c *Console) Warnf(format string, args ...any) {
	c.StopProgress()
	fmt.Fprintf(c.err, "WARN: "+format+"\n", args...)
}

func (c *Console) Errorf(format string, args ...any) {
	c.StopProgress()
	fmt.Fprintf(c.err, "ERROR: "+format+"\n", args...)
}

// ---------- Spinner (stderr)

func (c *Console) StartProgress(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.progressEnabled() {
		return
	}
	if c.sp == nil {
		c.sp = spinner.New(spinner.CharSets[69], 120*time.Millisecond, spinner.WithWriter(c.err))
	}
	c.sp.Suffix = " " + msg
	if !c.spOn {
		c.sp.Start()
		c.spOn = true
	}
}

func (c *Console) UpdateProgress(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.spOn || c.sp == nil {
		return
	}
	c.sp.Suffix = " " + msg
}

func (c *Console) StopProgress() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.spOn && c.sp != nil {
		c.sp.Stop()
		c.spOn = false
	}
}

func (c *Console) progressEnabled() bool {
	switch c.mode {
	case ProgressNever:
		return false
	case ProgressAlways:
		return isTerminal(c.err) && !inCI()
	default: // auto
		return isTerminal(c.err) && !inCI()
	}
}

func (c *Console) FinalizeProgress() {
	c.StopProgress()
	c.clearErrLine()
}

func (c *Console) clearErrLine() {
	fmt.Fprint(c.err, "\r\033[2K")
	fmt.Fprintf(c.out, "        \r") // windows override
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

func inCI() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("CI")))
	return v == "1" || v == "true" || v == "yes"
}
