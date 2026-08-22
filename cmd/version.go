package cmd

// Version is the current version of Model CLI
const Version = "0.1.0"

// Commit is the git commit hash (set at build time)
var Commit string

// Date is the build date (set at build time)
var Date string

// BuildInfo returns the version information
func GetVersion() string {
	if Commit != "" && Date != "" {
		return Version + " (commit: " + Commit + ", date: " + Date + ")"
	}
	return Version
}
