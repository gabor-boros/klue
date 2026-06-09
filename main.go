// Package main is the entrypoint for the klue CLI.
package main

import "github.com/gabor-boros/klue/cmd"

// Build information injected at release time via -ldflags. The default values
// are used for local (non-release) builds.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.Execute(version, commit, date)
}
