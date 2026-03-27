package main

import (
	"os"

	"socialpilot/cmd"
)

func main() {
	// Set embedded web UI filesystem
	if webUI, err := GetWebUIFS(); err == nil {
		cmd.SetWebUI(webUI)
	}
	os.Exit(cmd.Execute())
}
