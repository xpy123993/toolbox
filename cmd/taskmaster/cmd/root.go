package cmd

import (
	"context"
	"flag"
	"fmt"
	"time"
)

func HandleServe(args ...string) error {
	flagSet := flag.NewFlagSet("serve", flag.ExitOnError)
	snapshotInterval := flagSet.Duration("snapshot-interval", 30*time.Second, "Save interval of snapshots.")
	flagSet.Parse(args)
	if len(flagSet.Args()) != 2 {
		fmt.Println("Usage: serve [serving channel] [snapshot folder]")
		fmt.Println("Example: serve --snapshot-interval=30s /example/taskmaster ./snapshots")
		return fmt.Errorf("invalid arguments")
	}
	StartTaskMasterService(flagSet.Arg(0), flagSet.Arg(1), *snapshotInterval)
	return nil
}

func HandleWorker(args ...string) error {
	flagSet := flag.NewFlagSet("work", flag.ExitOnError)
	taskGroup := flagSet.String("task-group", "default", "Group this worker is assigned to.")
	taskTimeout := flagSet.Duration("task-timeout", time.Hour, "The timeout of executing each task.")
	flagSet.Parse(args)
	if len(flagSet.Args()) != 1 {
		fmt.Println("Usage: work [task master channel]")
		fmt.Println("Example: work /example/taskmaster --task-group=default --task-timeout=1h")
		return fmt.Errorf("invalid arguments")
	}
	StartWorker(flagSet.Arg(0), *taskGroup, *taskTimeout)
	return nil
}

func HandleInsert(args ...string) error {
	if len(args) < 3 {
		fmt.Println("Usage: insert [task master channel] [task group] [base command] [args ...]")
		fmt.Println("Example: insert /example/taskmaster echo hello world")
		return fmt.Errorf("invalid arguments")
	}
	InsertTask(context.Background(), args[0], args[1], args[2], args[3:])
	return nil
}
