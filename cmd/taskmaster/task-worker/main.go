package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os/exec"
	"time"

	"github.com/xpy123993/toolbox/pkg/task"
	"golang.org/x/net/trace"
)

var (
	taskMasterAddress = flag.String("task-master-address", "localhost:8080", "The address to access task master service.")
	defaultTimeout    = flag.Duration("execution-timeout", time.Hour, "Timeout to execute a task.")
	idleCycle         = flag.Duration("idle-cycle", time.Minute, "The cycle to pull a task if task master returns no task available.")

	statusAddress = flag.String("status-address", ":0", "The address to show the current status")
	finishCount   = 0
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
	tracker := trace.New("Worker", fmt.Sprintf("working on task `%s`", curTask.ID))
	defer tracker.Finish()
	log.Printf("working on task `%s`", curTask.ID)
	tracker.LazyPrintf("command: %s", curTask.Data)
	tracker.LazyPrintf("deadline: %v", curTask.AvailableTime)
	ctx, cancelFn := context.WithDeadline(context.Background(), curTask.AvailableTime.Add(-time.Second))
	defer cancelFn()
	cmd := exec.CommandContext(ctx, curShellTask.Base, curShellTask.Arguments...)
	if data, err := cmd.CombinedOutput(); err != nil {
		tracker.LazyPrintf(err.Error())
		tracker.SetError()
		return err
	} else {
		truncText := data
		if len(data) > 64 {
			truncText = data[:64]
		}
		tracker.LazyPrintf("Result: %s", truncText)
	}
	if resp, err = http.DefaultClient.Get(fmt.Sprintf("http://%s/finish?task_id=%s", *taskMasterAddress, curTask.ID)); err != nil || resp.StatusCode != 200 {
		if resp.StatusCode != 200 {
			data, _ := io.ReadAll(resp.Body)
			tracker.LazyPrintf(string(data))
			tracker.SetError()
			return fmt.Errorf("task master returns an error status (%d): %s", resp.StatusCode, data)
		}
		return err
	}
	finishCount++
	tracker.LazyPrintf("commited")
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

func startStatusService() {
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "Finish count: %d", finishCount)
	})
	listener, err := net.Listen("tcp", *statusAddress)
	if err != nil {
		log.Printf("failed to start status reporting service: %v", err)
		return
	}
	log.Printf("status page is available at %s", listener.Addr().String())
	http.Serve(listener, nil)
}

func main() {
	flag.Parse()
	go startStatusService()
	worker()
}
