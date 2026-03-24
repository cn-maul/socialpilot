package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"socialpilot/internal/config"
	"socialpilot/internal/exitcode"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Manage configuration"}
	cmd.AddCommand(newConfigSetCmd())
	return cmd
}

func newConfigSetCmd() *cobra.Command {
	var baseURL, apiKey, model, dbPath string
	var timeoutSec int
	c := &cobra.Command{
		Use:   "set",
		Short: "Set LLM and database config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, p, err := config.Load()
			if err != nil {
				return exitcode.New(exitcode.InvalidArg, "load config failed", err)
			}
			if strings.TrimSpace(baseURL) != "" {
				cfg.BaseURL = strings.TrimSpace(baseURL)
			}
			if strings.TrimSpace(apiKey) != "" {
				cfg.APIKey = strings.TrimSpace(apiKey)
			}
			if strings.TrimSpace(model) != "" {
				cfg.Model = strings.TrimSpace(model)
			}
			if strings.TrimSpace(dbPath) != "" {
				cfg.DBPath = strings.TrimSpace(dbPath)
			}
			if timeoutSec > 0 {
				cfg.TimeoutSeconds = timeoutSec
			}
			if err := config.Save(p, cfg); err != nil {
				return exitcode.New(exitcode.InvalidArg, "save config failed", err)
			}
			out := map[string]any{"status": "success", "config_path": p}
			if jsonOutput {
				return printJSON(out)
			}
			fmt.Printf("config saved: %s\n", p)
			return nil
		},
	}
	c.Flags().StringVar(&baseURL, "baseurl", "", "LLM base URL")
	c.Flags().StringVar(&apiKey, "apikey", "", "LLM API key")
	c.Flags().StringVar(&model, "model", "", "LLM model")
	c.Flags().StringVar(&dbPath, "db", "", "SQLite database path")
	c.Flags().IntVar(&timeoutSec, "timeout", 0, "LLM timeout seconds")
	return c
}
