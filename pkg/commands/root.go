package commands

import (
	"fmt"
	"os"
	"yoheiyayoi/bread/pkg/config"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Local
var rootCmd = &cobra.Command{
	Use:   config.AppName,
	Short: "🥖 Bread - Coolest Roblox package manager (v" + config.Version + ")",
}

var (
	LineBar   = "┃  "
	CheckIcon = color.GreenString("✓")
	InfoIcon  = color.BlueString("ℹ")
)

// Functions
func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
