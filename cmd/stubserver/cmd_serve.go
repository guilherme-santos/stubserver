package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/foodora/go-ranger/fdhttp"
	"github.com/guilherme-santos/stubserver"
	"github.com/guilherme-santos/stubserver/http"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

var (
	cfgFile string
	port    string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run http server",
	PreRun: func(cmd *cobra.Command, args []string) {
		portFlag := cmd.Flag("port")
		if !portFlag.Changed {
			if portEnv, ok := os.LookupEnv("STUBSERVER_PORT"); ok {
				port = portEnv
			}
		}

		if port == "" {
			cmd.Printf("Error: required flag \"%s\" or STUBSERVER_PORT envvar is not set\n", portFlag.Name)
			os.Exit(1)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		f, err := os.Open(cfgFile)
		if err != nil {
			cmd.Println(err)
			os.Exit(1)
		}

		defer f.Close()

		var cfg stubserver.Config

		err = yaml.NewDecoder(f).Decode(&cfg)
		if err != nil {
			cmd.Printf("Cannot read yaml file %s: %s\n", cfgFile, err)
			os.Exit(1)
		}

		router := fdhttp.NewRouter()

		logMiddleware := fdhttp.NewLogMiddleware()
		router.Use(logMiddleware.Middleware())

		handler := http.NewHandler(cfg)
		router.Register(handler)

		srv := fdhttp.NewServer(port)
		var errChan chan error
		go func() {
			errChan <- srv.Start(router)
		}()

		stopSignal := make(chan os.Signal, 2)
		signal.Notify(stopSignal, os.Interrupt, syscall.SIGTERM)

		// block until receive a SIGTERM or server.Start return
		select {
		case <-stopSignal:
			err := srv.Stop()
			if err != nil {
				log.Fatal("Cannot stop gracefully: ", err)
			}
		case err := <-errChan:
			log.Fatal("Cannot run http server: ", err)
		}

		log.Println("Stubserver stopped succesfuly!")
	},
}

func init() {
	serveCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "config file with spec of your stubs")
	serveCmd.Flags().StringVarP(&port, "port", "p", "80", "port to run the server or specify using STUBSERVER_PORT envvar")
	serveCmd.MarkFlagRequired("config")
	rootCmd.AddCommand(serveCmd)
}
