package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/xpy123993/toolbox/pkg/task"
)

var (
	taskMasterAddress = flag.String("task-master-address", "localhost:8080", "The address to access task master service.")
	defaultTimeout    = flag.Duration("execution-timeout", time.Hour, "Timeout to execute a task.")
	idleCycle         = flag.Duration("idle-cycle", time.Minute, "The cycle to pull a task if task master returns no task available.")
)

// ShellTask specifies the structure to execute a shell command.
type ShellTask struct {
	Base      string   `json:"base"`
	Arguments []string `json:"args"`
}

func workerRoutinue() error {
	resp, err := http.DefaultClient.Get(fmt.Sprintf("http://%s/query?timeout_sec=%d", *taskMasterAddress, int(defaultTimeout.Seconds())))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("no available task at present")
	}
	curTask := task.Task{}
	if err := json.NewDecoder(resp.Body).Decode(&curTask); err != nil {
		return err
	}
	curShellTask := ShellTask{}
	if err := json.Unmarshal([]byte(curTask.Data), &curShellTask); err != nil {
		return fmt.Errorf("unknown task format: %s", curTask.Data)
	}
	log.Printf("working on task `%s`", curTask.ID)
	ctx, cancelFn := context.WithDeadline(context.Background(), curTask.AvailableTime.Add(-time.Second))
	defer cancelFn()
	cmd := exec.CommandContext(ctx, curShellTask.Base, curShellTask.Arguments...)
	if _, err := cmd.CombinedOutput(); err != nil {
		return err
	}
	if resp, err = http.DefaultClient.Get(fmt.Sprintf("http://%s/finish?task_id=%s", *taskMasterAddress, curTask.ID)); err != nil || resp.StatusCode != 200 {
		if resp.StatusCode != 200 {
			data, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("task master returns an error status (%d): %s", resp.StatusCode, data)
		}
		return err
	}
	log.Printf("task `%s` is commited", curTask.ID)
	return nil
}

func worker() {
	for {
		if err := workerRoutinue(); err != nil {
			log.Printf("worker returns error status: %v", err)
			<-time.After(*idleCycle)
		}
	}
}

func main() {
	flag.Parse()
	worker()
}
