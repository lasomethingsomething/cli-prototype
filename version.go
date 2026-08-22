package main

// Version of the Model CLI
const Version = "0.1.0"

// BuildDate is the date when the binary was built
var BuildDate = ""

// GitCommit is the git commit hash used to build
var GitCommit = ""

// GitBranch is the git branch used to build
var GitBranch = ""

// GoVersion is the Go version used to build
var GoVersion = ""

// GetVersion returns the full version string
func GetVersion() string {
	version := "Model CLI version " + Version
	if GitCommit != "" {
		version += " (commit: " + GitCommit + ")"
	}
	if BuildDate != "" {
		version += " built on " + BuildDate
	}
	return version
}
