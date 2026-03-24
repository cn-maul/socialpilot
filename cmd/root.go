package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"socialpilot/internal/exitcode"
	"socialpilot/internal/service"
)

var jsonOutput bool

func Execute() int {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		type codeGetter interface{ ExitCode() int }
		var cg codeGetter
		if errors.As(err, &cg) {
			return cg.ExitCode()
		}
		return exitcode.InvalidArg
	}
	return exitcode.Success
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "socialpilot",
		Short:         "Local social CRM and AI copilot",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "output in JSON")

	root.AddCommand(newConfigCmd())
	root.AddCommand(newContactCmd())
	root.AddCommand(newLogCmd())
	root.AddCommand(newChatCmd())
	root.AddCommand(newCommitCmd())
	root.AddCommand(newAnalyzeCmd())
	root.AddCommand(newCompressCmd())
	root.AddCommand(newWebCmd())
	return root
}

func mustService(requireLLM bool) (*service.Service, func(), error) {
	svc, cleanup, err := service.OpenService(requireLLM)
	if err != nil {
		// Wrap errors with appropriate exit codes
		if strings.Contains(err.Error(), "llm config") {
			return nil, nil, exitcode.New(exitcode.InvalidArg, err.Error(), nil)
		}
		if strings.Contains(err.Error(), "database") {
			return nil, nil, exitcode.New(exitcode.Database, err.Error(), nil)
		}
		return nil, nil, exitcode.New(exitcode.InvalidArg, err.Error(), nil)
	}
	return svc, cleanup, nil
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func isJSONParseErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "cannot extract json") || strings.Contains(s, "invalid character") || strings.Contains(s, "unmarshal")
}

func wrapLLMErr(err error) error {
	if err == nil {
		return nil
	}
	if isJSONParseErr(err) {
		return exitcode.New(exitcode.LLMParse, "llm returned invalid json", err)
	}
	return exitcode.New(exitcode.NetworkErr, "llm request failed", err)
}

func wrapServiceErr(err error) error {
	if err == nil {
		return nil
	}
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "llm-parse:") || isJSONParseErr(err) {
		return exitcode.New(exitcode.LLMParse, "llm returned invalid json", err)
	}
	if strings.Contains(s, "llm:") {
		return exitcode.New(exitcode.NetworkErr, "llm request failed", err)
	}
	if strings.Contains(s, "no rows") || strings.Contains(s, "not found") || strings.Contains(s, "no messages") {
		return exitcode.New(exitcode.InvalidArg, "invalid request", err)
	}
	return exitcode.New(exitcode.Database, "database operation failed", err)
}
