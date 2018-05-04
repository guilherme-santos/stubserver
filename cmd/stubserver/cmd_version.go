package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// these var will be injected by our Makefile
var version = "0.0.0"
var build = "master"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Stubserver",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Stubserver version %s, build %s\n", version, build)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
