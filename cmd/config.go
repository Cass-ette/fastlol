package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"fastlol/internal"

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
	Long:  "Set a config value. Keys: rapidapi_key, riot_api_key, default_region",
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
	}
	if !validKeys[key] {
		internal.Error(fmt.Sprintf("Unknown config key: %s", key))
		fmt.Fprintln(os.Stderr, "Valid keys: rapidapi_key, riot_api_key, default_region")
		os.Exit(1)
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
		internal.Error(fmt.Sprintf("Failed to marshal config: %v", err))
		os.Exit(1)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		internal.Error(fmt.Sprintf("Failed to write config: %v", err))
		os.Exit(1)
	}

	fmt.Printf("Set %s = %s\n", key, maskKey(value))
	fmt.Printf("Config saved to %s\n", configPath)
}

func runConfigShow(cmd *cobra.Command, args []string) {
	internal.Title("Configuration")

	keys := []string{"rapidapi_key", "riot_api_key", "default_region"}
	for _, k := range keys {
		val := viper.GetString(k)
		if val == "" {
			val = "(not set)"
		} else if k == "rapidapi_key" || k == "riot_api_key" {
			val = maskKey(val)
		}
		fmt.Printf("  %-16s %s\n", k+":", val)
	}

	home, _ := os.UserHomeDir()
	fmt.Printf("\n  Config file: %s\n", filepath.Join(home, ".fastlol", "config.yaml"))
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
