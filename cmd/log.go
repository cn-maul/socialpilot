package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"socialpilot/internal/exitcode"
)

func newLogCmd() *cobra.Command {
	var name, message string
	c := &cobra.Command{
		Use:   "log",
		Short: "Ingest unstructured social log",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" || strings.TrimSpace(message) == "" {
				return exitcode.New(exitcode.InvalidArg, "--name and --message are required", nil)
			}
			svc, cleanup, err := mustService(true)
			if err != nil {
				return err
			}
			defer cleanup()

			sessionID, n, err := svc.IngestLog(name, message)
			if err != nil {
				return wrapServiceErr(err)
			}
			out := map[string]any{"status": "success", "session_id": sessionID, "inserted_count": n}
			if jsonOutput {
				return printJSON(out)
			}
			fmt.Printf("ingested %d messages into session %s\n", n, sessionID)
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "Contact name")
	c.Flags().StringVar(&message, "message", "", "Raw unstructured message")
	return c
}
