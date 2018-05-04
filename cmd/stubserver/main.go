package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "stubserver",
	Short: "Stubserver is a http server that will respond endpoint accordally with your config file",
	Long:  `Stubserver is a http server written in Go that helps you develop your app without need to have the real server running.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
