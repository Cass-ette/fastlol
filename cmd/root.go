package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "fastlol",
	Short: "Fast League of Legends CLI tool",
	Long:  "Query champion counters, builds, runes, tier lists, and summoner profiles from the terminal.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().String("rapidapi-key", "", "RapidAPI key (overrides config)")
}

func initConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	configDir := filepath.Join(home, ".fastlol")
	_ = os.MkdirAll(configDir, 0755)

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	viper.SetEnvPrefix("FASTLOL")
	viper.AutomaticEnv()

	_ = viper.ReadInConfig()

	// CLI flag overrides config file
	if key, _ := rootCmd.PersistentFlags().GetString("rapidapi-key"); key != "" {
		viper.Set("rapidapi_key", key)
	}
}
