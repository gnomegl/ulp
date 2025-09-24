package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "ulp",
	Short: "ULP - URL Line Processor for cleaning, deduplicating, and converting credential files",
	Long: `ULP (URL Line Processor) is a tool for processing credential files that:
- Cleans and normalizes domain formats from credential files
- Deduplicates entries with optional duplicate output file
- Creates NDJSON/JSONL files for Meilisearch indexing with freshness scoring
- Processes Telegram channel metadata when available
- Handles various input formats (URL:user:pass, domain:user:pass, etc.)
- Calculates freshness scores based on duplicate percentage and other factors`,
	Version: "2.0.1",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ulp.yaml)")
	rootCmd.PersistentFlags().IntVarP(&workers, "workers", "w", 0, "Number of worker threads (default: number of CPU cores)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress progress indicators and non-essential output")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".ulp")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
