package main

import (
	"os"
)

func preflight(s *settings) *settings {
	createIfMissing(s.log)
	createIfMissing(s.output)

	return s
}

func adjustSettingsBasedOnJobs(s *settings, numberOfJobs int) *settings {

	new := settings{}
	new = *s

	if new.maxThreads > numberOfJobs && numberOfJobs > 0 {
		new.maxThreads = numberOfJobs
	}

	return &new
}

func createIfMissing(path string) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
	}
}
