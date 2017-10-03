package main

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

type job struct {
	fileName string
	report   procReport
	settings *settings
	color    colorFunc
	logger   *logger
	errored  bool
	quit     chan bool
	done     chan bool
}

func do(j *job) {
	if j.settings.copyBeforeProc {
		err := copyBefore(j)
		if err != nil {
			j.errored = true
			j.logger.log(errors.New(logForJob(j)(err.Error())))
			j.done <- false
			close(j.done)
		}
	}

	var (
		outb bytes.Buffer
		errb bytes.Buffer
	)

	finalArgs := make([]string, len(j.settings.args), len(j.settings.args))
	for index, arg := range j.settings.args {
		if strings.Replace(arg, " ", "", -1) == "{{source}}" {
			finalArgs[index] = j.settings.source + j.fileName
		} else if strings.Replace(arg, " ", "", -1) == "{{output}}" {
			finalArgs[index] = j.settings.output + j.fileName
		} else {
			finalArgs[index] = arg
		}
	}

	cmd := exec.Command(j.settings.cmd, finalArgs...)

	cmd.Stdout = &outb
	cmd.Stderr = &errb

	j.logger.log(logForJob(j)("- start"))
	err := cmd.Start()
	if err != nil {
		j.logger.log(errors.New(logForJob(j)(err.Error())))
		j.errored = true
		j.done <- false
		close(j.done)
		return
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
		close(done)
	}()

	select {
	case <-j.quit:
		err := cmd.Process.Kill()
		if err != nil {
			j.logger.log(errors.New(logForJob(j)(err.Error())))
		}
		j.logger.log(logForJob(j)("- cancelled"))
		j.done <- false
		close(j.done)
		break
	case err := <-done:
		if err != nil {
			j.errored = true
			j.logger.log(errors.New(logForJob(j)(errb.String())))
			j.done <- false
			close(j.done)
		} else {
			j.logger.log(logForJob(j)(outb.String()))
			j.done <- true
			close(j.done)
		}
		break
	}
}
