package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"socialpilot/internal/exitcode"
)

func newCompressCmd() *cobra.Command {
	var all bool
	var name string
	c := &cobra.Command{
		Use:   "compress",
		Short: "Compress old sessions into summaries",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !all && strings.TrimSpace(name) == "" {
				return exitcode.New(exitcode.InvalidArg, "use --all or --name", nil)
			}
			svc, cleanup, err := mustService(true)
			if err != nil {
				return err
			}
			defer cleanup()

			n, err := svc.Compress(all, name)
			if err != nil {
				return wrapServiceErr(err)
			}
			out := map[string]any{"status": "success", "compressed_count": n}
			if jsonOutput {
				return printJSON(out)
			}
			fmt.Printf("compressed sessions: %d\n", n)
			return nil
		},
	}
	c.Flags().BoolVar(&all, "all", false, "compress for all contacts")
	c.Flags().StringVar(&name, "name", "", "compress for one contact")
	return c
}
