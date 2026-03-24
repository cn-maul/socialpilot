package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"socialpilot/internal/exitcode"
)

func newCommitCmd() *cobra.Command {
	var name, message string
	c := &cobra.Command{
		Use:   "commit",
		Short: "Record selected/sent user reply",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" || strings.TrimSpace(message) == "" {
				return exitcode.New(exitcode.InvalidArg, "--name and --message are required", nil)
			}
			svc, cleanup, err := mustService(false)
			if err != nil {
				return err
			}
			defer cleanup()

			sid, err := svc.CommitMessage(name, message)
			if err != nil {
				return exitcode.New(exitcode.Database, "commit failed", err)
			}
			out := map[string]any{"status": "success", "session_id": sid}
			if jsonOutput {
				return printJSON(out)
			}
			fmt.Printf("committed into session %s\n", sid)
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "Contact name")
	c.Flags().StringVar(&message, "message", "", "User final message")
	return c
}
