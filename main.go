package main

import "github.com/yarlson/lnk/cmd"

// These variables are set by GoReleaser via ldflags
var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	cmd.SetVersion(version, buildTime)
	cmd.Execute()
}
