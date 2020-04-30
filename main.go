package main

import "github.com/lnxjedi/gopherbot/bot"

// Version of gopherbot
var Version = "v2.0.0-beta3-snapshot"

// Commit supplied during linking
var Commit = "(not set)"

func main() {
	versionInfo := bot.VersionInfo{
		Version: Version,
		Commit:  Commit,
	}
	bot.Start(versionInfo)
}
