package flags

import "github.com/spf13/cobra"

type CommonFlags struct {
	JsonFile    string
	ChannelName string
	ChannelAt   string
	OutputDir   string
	Split       bool
	NoFreshness bool
	DupesFile   string
	NoDedupe    bool
}

func AddTelegramFlags(cmd *cobra.Command, flags *CommonFlags) {
	cmd.Flags().StringVarP(&flags.JsonFile, "json-file", "j", "", "Path to Telegram JSON export file (auto-detected if in same directory)")
	cmd.Flags().StringVarP(&flags.ChannelName, "channel-name", "n", "", "Telegram channel name")
	cmd.Flags().StringVarP(&flags.ChannelAt, "channel-at", "a", "", "Telegram channel @username")
}

func AddOutputFlags(cmd *cobra.Command, flags *CommonFlags) {
	cmd.Flags().StringVarP(&flags.OutputDir, "output-dir", "o", "", "Output directory for generated files")
	cmd.Flags().BoolVarP(&flags.Split, "split", "s", false, "Split output files at 100MB")
	cmd.Flags().BoolVar(&flags.NoFreshness, "no-freshness", false, "Disable freshness scoring")
}

func AddDedupeFlags(cmd *cobra.Command, flags *CommonFlags) {
	cmd.Flags().StringVar(&flags.DupesFile, "dupes-file", "", "Path to save duplicate entries")
	cmd.Flags().BoolVar(&flags.NoDedupe, "no-dedupe", false, "Disable deduplication")
}

func AddAllFlags(cmd *cobra.Command, flags *CommonFlags) {
	AddTelegramFlags(cmd, flags)
	AddOutputFlags(cmd, flags)
	AddDedupeFlags(cmd, flags)
}