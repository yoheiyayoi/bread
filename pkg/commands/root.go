package commands

import (
	"fmt"
	"os"
	"yoheiyayoi/bread/pkg/config"
	"yoheiyayoi/bread/pkg/utils"

	"github.com/charmbracelet/log"
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

	// update checker
	isLatest, latestVer, err := utils.CheckForUpdates()
	if err != nil {
		log.Error(err)
		return
	}

	if !isLatest {
		info := color.New(color.FgCyan, color.Bold).SprintFunc()

		fmt.Println(color.YellowString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
		fmt.Printf("%s A new version of Bread is available: %s → %s\n", InfoIcon, color.RedString(config.Version), color.GreenString(latestVer))
		fmt.Printf("%s To update, run: %s\n", InfoIcon, info("bread self-update"))
		fmt.Println(color.YellowString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
