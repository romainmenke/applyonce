package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
)

func readSettings() *settings {
	buf := bytes.NewBuffer(nil)
	file, err := os.Open("applyonce.conf")
	if err != nil {
		return nil
	}

	_, err = io.Copy(buf, file)
	if err != nil {
		return nil
	}

	file.Close()
	sf := &settingsFile{}

	err = json.Unmarshal(buf.Bytes(), sf)
	if err != nil {
		return nil
	}

	s := &settings{
		logLevel:       sf.LogLevel,
		force:          sf.Force,
		log:            sf.LogDirectory,
		output:         sf.OutputDirectory,
		source:         sf.SourceDirectory,
		maxThreads:     sf.MaxThreads,
		dontGrow:       sf.DontGrow,
		copy:           sf.Copy,
		copyBeforeProc: sf.CopyBeforeProc,
		interval:       sf.Interval,
		cmd:            sf.Cmd,
		args:           sf.Args,
	}

	s.log = strings.TrimSuffix(s.log, "/") + "/"
	s.output = strings.TrimSuffix(s.output, "/") + "/"
	s.source = strings.TrimSuffix(s.source, "/") + "/"

	return s
}

type settingsFile struct {
	LogLevel        string   `json:"logLevel"`
	Force           bool     `json:"force"`
	LogDirectory    string   `json:"logDirectory"`
	OutputDirectory string   `json:"outputDirectory"`
	SourceDirectory string   `json:"sourceDirectory"`
	MaxThreads      int      `json:"maxThreads"`
	DontGrow        bool     `json:"dontGrow"`
	Copy            bool     `json:"copy"`
	CopyBeforeProc  bool     `json:"copyBeforeProc"`
	Interval        int      `json:"interval"`
	Cmd             string   `json:"cmd"`
	Args            []string `json:"args"`
}

var example = `{
	"logLevel": "info",
	"force": false,
	"logDirectory": "./log",
	"outputDirectory": "./output",
	"sourceDirectory": "./source",
	"maxThreads": 3,
	"dontGrow": true,
	"copy": false,
	"copyBeforeProc": false,
	"interval": 30,
	"cmd": "echo",
	"args": [
		"{{ source }}",
		">",
		"{{ output }}",
	]
}
`
