package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mgutz/ansi"
)

func main() {

	settings := parseArgs()
	if settings == nil {
		return
	}

	if settings.logLevel == "info" || settings.logLevel == "debug" {
		fmt.Printf("Source        =>  %s\n", settings.source)
		fmt.Printf("Output        =>  %s\n", settings.output)
		fmt.Printf("Log           =>  %s\n", settings.log)
		fmt.Printf("Force         =>  %t\n", settings.force)
		fmt.Printf("Threads       =>  %d\n", settings.maxThreads)
		fmt.Printf("Dont Grow     =>  %t\n", settings.dontGrow)
		fmt.Printf("Copy          =>  %t\n", settings.copy)
		fmt.Printf("Copy Before   =>  %t\n", settings.copy)
		fmt.Printf("Interval      =>  %d\n", settings.interval)
		fmt.Printf("Cmd           =>  %s\n", settings.cmd+" "+strings.Join(settings.args, " "))
	}

	logger := newLogger(settings.logLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan struct{})

	var ticker *time.Ticker

	if settings.interval > 0 {
		ticker = time.NewTicker(time.Second * time.Duration(settings.interval))
	}

	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		_ = <-signalChannel
		if ticker != nil {
			ticker.Stop()
		}

		cancel()
		time.Sleep(time.Second)
		close(quit)
	}()

	if settings.interval > 0 {
		for {
			select {
			case <-ticker.C:
				runOnce(ctx, settings, logger)
			case <-quit:
				return
			}
		}
	} else {
		runOnce(ctx, settings, logger)
	}
}

func runOnce(ctx context.Context, settings *settings, logger *logger) {

	var (
		reports    map[string]procReport
		newReports map[string]procReport
		jobs       []*job
	)

	reports = getReports(settings.log)
	jobs, newReports = getJobs(settings, reports, logger)

	if len(jobs) == 0 {
		return
	}

	settings = adjustSettingsBasedOnJobs(settings, len(jobs))

	if settings.force {
		saveReports(newReports, settings.log)
	}

	jobsQueue := make(chan *job, len(jobs))
	for _, j := range jobs {
		jobsQueue <- j
	}
	close(jobsQueue)

	var wg sync.WaitGroup
	var cancels []chan bool
	for i := 0; i < settings.maxThreads; i++ {
		wg.Add(1)

		go func() {
		JOB_QUEUE:
			for j := range jobsQueue {

				cancel := make(chan bool, 1)
				cancels = append(cancels, cancel)

				go func() {
					do(j)
				}()

				select {
				case success := <-j.done:
					if success {
						newReports[j.report.Path] = j.report
					}

					after(j)
					time.Sleep(time.Millisecond * 10)

				case <-cancel:
					close(j.quit)
					break JOB_QUEUE
				}
			}
			wg.Done()
		}()
	}

	go func() {
		<-ctx.Done()
		for _, c := range cancels {
			close(c)
		}
	}()

	wg.Wait()

	saveReports(newReports, settings.log)
}

func getJobs(settings *settings, reports map[string]procReport, logger *logger) ([]*job, map[string]procReport) {

	newReports := make(map[string]procReport)

	files, err := ioutil.ReadDir(settings.source)
	if err != nil {
		panic(err)
	}

	jobs := []*job{}
FILE_ITERATOR:
	for index, f := range files {

		if !isFile(settings.source + f.Name()) {
			continue FILE_ITERATOR
		}

		if strings.HasPrefix(f.Name(), ".") {
			continue FILE_ITERATOR
		}

		j := &job{
			fileName: f.Name(),
			report:   reports[settings.source+f.Name()],
			settings: settings,
			quit:     make(chan bool, 1),
			done:     make(chan bool, 1),
			logger:   logger,
		}

		j.color = ansi.ColorFunc(colors[index%len(colors)])

		if !needsProc(j) {
			newReports[j.report.Path] = j.report
			fmt.Printf("%s %s : \n- skipped\n", time.Now().Format("15:04:05"), j.settings.source+j.fileName)
			continue FILE_ITERATOR
		}

		jobs = append(jobs, j)
	}

	return jobs, newReports
}

func after(j *job) {

	if j.settings.dontGrow {
		err := preventGrow(j)
		if err != nil {
			j.logger.log(errors.New(logForJob(j)(err.Error())))
		}
	}

	if j.settings.copy && !j.settings.copyBeforeProc {
		err := copyAfter(j)
		if err != nil {
			j.logger.log(errors.New(logForJob(j)(err.Error())))
		}
	}
}

func preventGrow(j *job) error {

	if !isFile(j.settings.source + j.fileName) {
		return nil
	}
	if !isFile(j.settings.output + j.fileName) {
		return nil
	}

	originalInfo, err := os.Stat(j.settings.source + j.fileName)
	if err != nil {
		return err
	}

	resultInfo, err := os.Stat(j.settings.output + j.fileName)
	if err != nil {
		return err
	}

	if originalInfo.Size() > resultInfo.Size() {
		return nil
	}

	j.logger.log(logForJob(j)(fmt.Sprintf("- grew form %dkb to %dkb", (originalInfo.Size() / 1024), (resultInfo.Size() / 1024))))

	if !j.settings.dontGrow {
		return nil
	}

	j.errored = true
	j.logger.log(logForJob(j)("- deleting the oversized file"))

	err = os.Remove(j.settings.output + j.fileName)
	if err != nil {
		return err
	}

	return nil

}

func copyBefore(j *job) error {

	var (
		err          error
		originalFile *os.File
		resultFile   *os.File
	)

	if strings.HasPrefix(j.fileName, ".") {
		return nil
	}
	if !isFile(j.settings.source + j.fileName) {
		return nil
	}

	if exists(j.settings.output+j.fileName) && sha1ForFile(j.settings.output+j.fileName) == sha1ForFile(j.settings.source+j.fileName) {
		return nil
	}

	originalFile, err = os.Open(j.settings.source + j.fileName)
	if err != nil {
		return err
	}
	defer originalFile.Close()

	resultFile, err = os.Create(j.settings.output + j.fileName)
	if err != nil {
		return err
	}
	defer resultFile.Close()

	_, err = io.Copy(resultFile, originalFile)
	if err != nil {
		return err
	}

	err = resultFile.Sync()
	if err != nil {
		return err
	}

	j.logger.log(logForJob(j)("- copied"))

	return nil
}

func copyAfter(j *job) error {

	// must have errored
	if !j.errored {
		return nil
	}

	var (
		err          error
		originalFile *os.File
		resultFile   *os.File
	)

	if strings.HasPrefix(j.fileName, ".") {
		return nil
	}
	if !isFile(j.settings.source + j.fileName) {
		return nil
	}

	if exists(j.settings.output+j.fileName) && sha1ForFile(j.settings.output+j.fileName) == sha1ForFile(j.settings.source+j.fileName) {
		return nil
	}

	originalFile, err = os.Open(j.settings.source + j.fileName)
	if err != nil {
		return err
	}
	defer originalFile.Close()

	resultFile, err = os.Create(j.settings.output + j.fileName)
	if err != nil {
		return err
	}
	defer resultFile.Close()

	_, err = io.Copy(resultFile, originalFile)
	if err != nil {
		return err
	}

	err = resultFile.Sync()
	if err != nil {
		return err
	}

	j.logger.log(logForJob(j)("- copied"))

	return nil
}
