package main

import (
	"fmt"
	"runtime"
	"strings"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var defaultMaxThreads = 3

type argList []string

func (l *argList) Set(value string) error {
	*l = append(*l, value)
	return nil
}

func (l *argList) String() string {
	return ""
}

func (l *argList) IsCumulative() bool {
	return true
}

func args(s kingpin.Settings) *[]string {
	target := new([]string)
	s.SetValue((*argList)(target))
	return target
}

func parseArgs() *settings {

	s := readSettings()
	if s != nil {
		return s
	}

	logLevel := kingpin.Flag("log", "Log level").
		Default("info").
		String()

	force := kingpin.Flag("force", "Force reprocessing").
		Short('f').
		Bool()

	maxThreads := kingpin.Flag("threads", fmt.Sprintf("Max concurrent threads. Default limit is %d", maxParallelism()-1)).
		Short('t').
		Default(fmt.Sprint(defaultMaxThreads)).
		Uint()

	dontGrow := kingpin.Flag("dontgrow", "Delete compressed files that got bigger").
		Short('g').
		Bool()

	copy := kingpin.Flag("copy", "Copy all files from source folder to output, without overwriting succesful results").
		Short('c').
		Bool()

	copyBeforeProc := kingpin.Flag("copy before proc", "Copy all files from source folder to output, before running cmd").
		Bool()

	interval := kingpin.Flag("interval", "Process interval").
		Short('i').
		Int()

	source := kingpin.Arg("source", "Source directory").
		Default("./").
		String()

	output := kingpin.Arg("output", "Output directory").
		Default("./").
		String()

	log := kingpin.Arg("log", "Log directory, the log is used to prevent duplicate handling").
		Default("").
		String()

	cmd := kingpin.Arg("cmd", "Command").
		String()

	args := args(kingpin.Arg("args", "Arguments"))

	kingpin.Parse()

	if *log == "" {
		*log = *output
	}

	*log = strings.TrimSuffix(*log, "/") + "/"
	*output = strings.TrimSuffix(*output, "/") + "/"
	*source = strings.TrimSuffix(*source, "/") + "/"

	s = &settings{
		logLevel: *logLevel,

		force: *force,

		log:    *log,
		output: *output,
		source: *source,

		maxThreads: int(*maxThreads),

		dontGrow:       *dontGrow,
		copy:           *copy,
		copyBeforeProc: *copyBeforeProc,

		interval: *interval,

		cmd:  *cmd,
		args: *args,
	}

	return s
}

type settings struct {
	logLevel string

	force bool

	log    string
	output string
	source string

	maxThreads int

	dontGrow       bool
	copy           bool
	copyBeforeProc bool

	interval int

	cmd  string
	args []string
}

func maxParallelism() int {
	maxProcs := runtime.GOMAXPROCS(0)
	numCPU := runtime.NumCPU()
	if maxProcs < numCPU {
		return maxProcs
	}
	return numCPU
}
