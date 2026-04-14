package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"fastlol/internal"
	"fastlol/internal/i18n"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage fastlol configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Long:  "Set a config value. Keys: rapidapi_key, riot_api_key, default_region, language, api_url",
	Args:  cobra.ExactArgs(2),
	Run:   runConfigSet,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run:   runConfigShow,
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigSet(cmd *cobra.Command, args []string) {
	key, value := args[0], args[1]

	validKeys := map[string]bool{
		"rapidapi_key":   true,
		"riot_api_key":   true,
		"default_region": true,
		"language":       true,
		"api_url":        true,
	}
	if !validKeys[key] {
		internal.Error(fmt.Sprintf(i18n.T("error.unknown_key"), key))
		fmt.Fprintln(os.Stderr, "  "+i18n.T("error.valid_keys"))
		os.Exit(1)
	}

	// Validate language value
	if key == "language" {
		validLangs := map[string]bool{"en": true, "zh": true, "ko": true}
		if !validLangs[value] {
			internal.Error(fmt.Sprintf(i18n.T("lang.invalid"), value))
			os.Exit(1)
		}
	}

	viper.Set(key, value)

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".fastlol", "config.yaml")

	// Read existing config or create new
	config := make(map[string]interface{})
	for _, k := range viper.AllKeys() {
		config[k] = viper.Get(k)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		internal.Error(fmt.Sprintf(i18n.T("error.marshal_config"), err))
		os.Exit(1)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		internal.Error(fmt.Sprintf(i18n.T("error.config_write"), err))
		os.Exit(1)
	}

	fmt.Printf("  " + i18n.T("title.set") + "\n", key, maskKey(value))
	fmt.Printf("  " + i18n.T("title.saved") + "\n", configPath)

	// Reload config so language change takes effect immediately
	viper.SetConfigFile(configPath)
	viper.ReadInConfig()

	// Show current language after setting
	if key == "language" {
		lang := viper.GetString("language")
		langName := langDisplayName(lang)
		fmt.Printf("\n  " + i18n.T("lang.current") + "\n", langName)
	}
}

func langDisplayName(lang string) string {
	switch lang {
	case "zh":
		return i18n.T("lang.zh")
	case "ko":
		return i18n.T("lang.ko")
	default:
		return i18n.T("lang.en")
	}
}

func runConfigShow(cmd *cobra.Command, args []string) {
	internal.Title(i18n.T("title.config"))

	keys := []string{"rapidapi_key", "riot_api_key", "default_region", "language", "api_url"}
	for _, k := range keys {
		val := viper.GetString(k)
		if val == "" {
			val = i18n.T("title.not_set")
		} else if k == "rapidapi_key" || k == "riot_api_key" {
			val = maskKey(val)
		} else if k == "language" {
			val = langDisplayName(val)
		}
		fmt.Printf("  %-16s %s\n", k+":", val)
	}

	home, _ := os.UserHomeDir()
	fmt.Printf("\n  " + i18n.T("tip.config_file") + "\n", filepath.Join(home, ".fastlol", "config.yaml"))
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
