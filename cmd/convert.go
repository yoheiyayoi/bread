package cmd

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

// For anyone who read this
// I don't recomment to convert to wally if you use custom package directories in bread.toml
// because wally doesn't support that feature yet
// and if you already used it, you have to manually rename the folders or re-install packages after converting
// in case if you use and you rename the folders to wally style, the script that requires those packages will break!!

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Create wally.toml from bread.toml",
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat("wally.toml"); err == nil {
			log.Error("wally.toml already exists!")
			return
		}

		var data map[string]any
		if _, err := toml.DecodeFile("bread.toml", &data); err != nil {
			log.Error("Failed to parse bread.toml:", err)
			return
		}

		// Remove the [bread] section
		delete(data, "bread")

		file, err := os.Create("wally.toml")
		if err != nil {
			log.Error("Failed to create wally.toml:", err)
			return
		}
		defer file.Close()

		if err := toml.NewEncoder(file).Encode(data); err != nil {
			log.Error("Failed to write to wally.toml:", err)
			return
		}

		log.Info("Successfully created wally.toml")
		log.Warn("Wally doesn't support custom package directories. If you're using a custom directory in Bread, please run wally install or rename the folder.")
	},
}

func init() {
	rootCmd.AddCommand(convertCmd)
}
