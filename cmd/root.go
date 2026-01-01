package cmd

import (
	"fmt"
	"os"
	"yoheiyayoi/bread/config"

	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Masterminds/semver/v3"
)

var rootCmd = &cobra.Command{
	Use:   config.AppName,
	Short: "Bread CLI tool 🥖 - v" + config.Version,
}

const (
	currentVersion = config.Version
	repoOwner      = config.RepoOwner
	repoName       = config.RepoName
	checkInterval  = 24 * time.Hour
)

type UpdateChecker struct {
	cacheFile string
}

type CacheData struct {
	LastCheck    time.Time `json:"last_check"`
	LatestVer    string    `json:"latest_version"`
	ChangelogURL string    `json:"changelog_url"`
}

// Functions
func NewUpdateChecker() *UpdateChecker {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".bread")
	os.MkdirAll(cacheDir, 0755)

	return &UpdateChecker{
		cacheFile: filepath.Join(cacheDir, "update_cache.json"),
	}
}

func (uc *UpdateChecker) loadCache() (*CacheData, error) {
	data, err := os.ReadFile(uc.cacheFile)
	if err != nil {
		return nil, err
	}

	var cache CacheData
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

func (uc *UpdateChecker) saveCache(cache *CacheData) error {
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(uc.cacheFile, data, 0644)
}

func (uc *UpdateChecker) shouldCheck() bool {
	cache, err := uc.loadCache()
	if err != nil {
		return true
	}

	return time.Since(cache.LastCheck) > checkInterval
}

func (uc *UpdateChecker) fetchLatestVersion() (string, string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", err
	}

	return strings.TrimPrefix(release.TagName, "v"), release.HTMLURL, nil
}

func (uc *UpdateChecker) CheckForUpdates() {
	if !uc.shouldCheck() {
		cache, _ := uc.loadCache()
		if cache != nil {
			uc.displayIfUpdateAvailable(cache.LatestVer, cache.ChangelogURL)
			return
		}
	}

	latestVer, changelogURL, err := uc.fetchLatestVersion()
	if err != nil {
		return
	}

	cache := &CacheData{
		LastCheck:    time.Now(),
		LatestVer:    latestVer,
		ChangelogURL: changelogURL,
	}
	uc.saveCache(cache)

	uc.displayIfUpdateAvailable(latestVer, changelogURL)
}

func (uc *UpdateChecker) displayIfUpdateAvailable(latestVer, changelogURL string) {
	current, err := semver.NewVersion(currentVersion)
	if err != nil {
		return
	}

	latest, err := semver.NewVersion(latestVer)
	if err != nil {
		return
	}

	linebar := color.YellowString("┃  ")

	if latest.GreaterThan(current) {
		// info := color.New(color.FgCyan, color.Bold).SprintFunc()

		fmt.Printf("%s%s %s → %s\n", linebar, color.YellowString("update available!"), color.RedString(currentVersion), color.GreenString(latestVer))
		fmt.Printf("%sgo to %s to download\n", linebar, color.CyanString(changelogURL)) // bread self-upgrade not implemented yet bruh
		fmt.Println()

		// fmt.Printf("%srun %s to upgrade\n", linebar, info("`bread self-upgrade`"))
		// fmt.Printf("%schangelog: %s\n", linebar, changelogURL)
		// fmt.Printf("%s\n", linebar)
	}
}

func init() {
	checker := NewUpdateChecker()
	checker.CheckForUpdates()

	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
