package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"socialpilot/internal/exitcode"
)

func newAnalyzeCmd() *cobra.Command {
	var name string
	c := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze contact personality/profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" {
				return exitcode.New(exitcode.InvalidArg, "--name is required", nil)
			}
			svc, cleanup, err := mustService(true)
			if err != nil {
				return err
			}
			defer cleanup()

			summary, err := svc.AnalyzeContact(name)
			if err != nil {
				return wrapServiceErr(err)
			}
			if jsonOutput {
				return printJSON(map[string]any{"status": "success", "profile_summary": summary})
			}
			fmt.Println(summary)
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "Contact name")
	return c
}
