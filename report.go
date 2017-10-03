package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"time"
)

type procReport struct {
	ModTime time.Time `json:"revisionTime"`
	Path    string    `json:"path"`
	Sha1    string    `json:"checksumSHA1"`
	Cmd     string    `json:"cmd"`
}

func (r procReport) Empty() bool {
	return r.ModTime.IsZero() && r.Path == "" && r.Sha1 == "" && r.Cmd == ""
}

func getReports(path string) map[string]procReport {
	buf := bytes.NewBuffer(nil)
	file, err := os.Open(path + "log.json")
	if err != nil {
		return make(map[string]procReport)
	}

	_, err = io.Copy(buf, file)
	if err != nil {
		return make(map[string]procReport)
	}

	file.Close()
	reports := make(map[string]procReport)

	err = json.Unmarshal(buf.Bytes(), &reports)
	if err != nil {
		return make(map[string]procReport)
	}

	return reports
}

func saveReports(reports map[string]procReport, path string) {
	b, err := json.Marshal(reports)
	if err != nil {
		panic(err)
	}

	var out bytes.Buffer
	err = json.Indent(&out, b, "", "  ")
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(path+"log.json", out.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
}

func timeModified(filePath string) time.Time {
	info, err := os.Stat(filePath)
	if err != nil {
		panic(err)
	}
	return info.ModTime()
}

func sha1ForFile(filePath string) string {
	var returnSHA1String string
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	defer file.Close()
	hash := sha1.New()
	if _, err = io.Copy(hash, file); err != nil {
		panic(err)
	}

	hashInBytes := hash.Sum(nil)[:20]
	returnSHA1String = hex.EncodeToString(hashInBytes)

	return returnSHA1String
}

func writeFile(content []byte, fileName string, out string, quality int) {
	err := ioutil.WriteFile(out+fileName, content, 0644)
	if err != nil {
		panic(err)
	}
}

func isFile(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !fileInfo.IsDir()
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return !os.IsNotExist(err)
	}
	return true
}
