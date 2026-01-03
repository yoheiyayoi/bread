package main

import (
	"yoheiyayoi/bread/pkg/commands"

	"github.com/charmbracelet/log"
)

// Functions
func main() {
	log.SetReportTimestamp(false)
	commands.Execute()
}
