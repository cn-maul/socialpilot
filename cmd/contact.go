package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"socialpilot/internal/exitcode"
	"socialpilot/internal/service"
)

func newContactCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "contact", Short: "Contact operations"}
	cmd.AddCommand(newContactAddCmd())
	return cmd
}

func newContactAddCmd() *cobra.Command {
	var name, gender, tags string
	c := &cobra.Command{
		Use:   "add",
		Short: "Add a contact",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" {
				return exitcode.New(exitcode.InvalidArg, "--name is required", nil)
			}
			svc, cleanup, err := mustService(false)
			if err != nil {
				return err
			}
			defer cleanup()

			contact, err := svc.AddContact(service.ContactInput{Name: name, Gender: gender, Tags: tags})
			if err != nil {
				return exitcode.New(exitcode.Database, "add contact failed", err)
			}
			out := map[string]any{"status": "success", "id": contact.ID, "name": contact.Name}
			if jsonOutput {
				return printJSON(out)
			}
			fmt.Printf("contact added: %s (%s)\n", contact.Name, contact.ID)
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "Contact name")
	c.Flags().StringVar(&gender, "gender", "unknown", "male/female/other/unknown")
	c.Flags().StringVar(&tags, "tags", "", "comma-separated tags")
	return c
}
