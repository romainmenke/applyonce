package main

func needsProc(j *job) bool {

	var changed bool
	var settingsChanged bool

	if j.report.Empty() || j.settings.force {
		settingsChanged = true
		changed = true
	}

	if j.report.Cmd != j.settings.cmd {
		settingsChanged = true
		changed = true
	}

	modTime := timeModified(j.settings.source + j.fileName)
	if !j.report.Empty() && !modTime.Equal(j.report.ModTime) {
		changed = true
	}

	sha := sha1ForFile(j.settings.source + j.fileName)
	if !j.report.Empty() && sha != j.report.Sha1 {
		changed = true
	}

	if !changed && !settingsChanged {
		return false
	}

	j.report.Cmd = j.settings.cmd
	j.report.ModTime = modTime
	j.report.Path = j.settings.source + j.fileName
	j.report.Sha1 = sha

	return true

}
