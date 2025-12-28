package cmd

import (
	"fmt"
	"os"
	"yoheiyayoi/bread/config"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   config.AppName,
	Short: "Bread CLI tool 🥖 - v" + config.Version,
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
