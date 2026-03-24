package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"socialpilot/internal/exitcode"
)

func newChatCmd() *cobra.Command {
	var name, message string
	c := &cobra.Command{
		Use:   "chat",
		Short: "Generate reply suggestions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" || strings.TrimSpace(message) == "" {
				return exitcode.New(exitcode.InvalidArg, "--name and --message are required", nil)
			}
			svc, cleanup, err := mustService(true)
			if err != nil {
				return err
			}
			defer cleanup()

			advice, sessionID, err := svc.ChatAdvice(name, message)
			if err != nil {
				return wrapServiceErr(err)
			}
			out := map[string]any{"session_id": sessionID, "advice": advice}
			if jsonOutput {
				return printJSON(out)
			}
			for i, a := range advice {
				fmt.Printf("%d. [%s] %s\n", i+1, a.Tone, a.Content)
			}
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "Contact name")
	c.Flags().StringVar(&message, "message", "", "Incoming message from contact")
	return c
}
