package cmd

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/xpy123993/yukino-net/libraries/util"
	"golang.org/x/net/trace"
)

var (
	exporterAddress string
	netConfig       *util.ClientConfig
	configFile      = []string{}

	rootCmd = &cobra.Command{
		Use:   "taskmaster",
		Short: "taskmaster is a tool to distribute tasks to workers.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if len(exporterAddress) > 0 {
				go func() {
					lis, err := net.Listen("tcp", exporterAddress)
					if err != nil {
						log.Printf("failed to start status exporter: %v", err)
						return
					}
					log.Printf("Exporting metrics on http://%s/debug/requests", lis.Addr().String())
					http.Serve(lis, nil)
				}()
			}
		},
	}
)

func loadNetConfig(cmd *cobra.Command, args []string) error {
	var err error
	netConfig, err = util.LoadNetConfig(configFile)
	if err != nil {
		fmt.Printf("Cannot load yukino net configuration file: %v\n", err)
	}
	return err
}

// Execute executes commands from os.Args.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("failed to execute Root command: %v", err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringArrayVarP(&configFile, "config", "c", []string{"./config.json", "/etc/yukino-net/config.json", "$HOME/.yukino-net"}, "Configuration file to join the router network.")
	rootCmd.PersistentFlags().StringVarP(&exporterAddress, "exporter-address", "e", "", "The webserver to export realtime metrics.")

	trace.AuthRequest = func(req *http.Request) (any, sensitive bool) {
		return true, false
	}

	snapshotInterval := time.Minute

	var serveCmd = &cobra.Command{
		Use:     "serve [serving channel] [snapshot folder]",
		Short:   "Starts a task master service.",
		Example: "serve /example/taskmaster ./snapshots --snapshot-interval=30s",
		PreRunE: loadNetConfig,
		Args:    cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			StartTaskMasterService(netConfig, args[0], args[1], snapshotInterval)
		},
	}
	serveCmd.Flags().DurationVarP(&snapshotInterval, "snapshot-interval", "t", time.Minute, "The save interval of snapshots.")

	taskGroup := "default"
	taskTimeout := time.Hour

	var workCmd = &cobra.Command{
		Use:     "work [task master channel]",
		Short:   "Starts a worker job to fetch tasks from task master channel.",
		Example: "work /example/taskmaster --task-group=default --task-timeout=1h",
		PreRunE: loadNetConfig,
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			StartWorker(netConfig, args[0], taskGroup, taskTimeout)
		},
	}
	workCmd.Flags().StringVarP(&taskGroup, "task-group", "g", "default", "The group of the task to be fetched.")
	workCmd.Flags().DurationVarP(&taskTimeout, "task-timeout", "t", time.Hour, "The timeout of executing each task.")

	var insertCmd = &cobra.Command{
		Use:     "insert [task master channel] [task group] [base command] [args ...]",
		Short:   "Insert a task into task master channel",
		Example: "insert /example/taskmaster echo hello world",
		PreRunE: loadNetConfig,
		Args:    cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			InsertTask(cmd.Context(), netConfig, args[0], args[1], args[2], args[3:])
		},
	}

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(workCmd)
	rootCmd.AddCommand(insertCmd)
}
