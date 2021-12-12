package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/xpy123993/toolbox/pkg/task"
)

var (
	taskMaster *task.Scheduler

	snapshotFile       = flag.String("snapshot-file", "snapshot.json", "The snapshot file of the task master.")
	snapshotInterval   = flag.Duration("snapshot-interval", time.Minute, "The snapshot interval.")
	taskDefaultTimeout = flag.Duration("query-default-timeout", time.Hour, "The default duration of a task to be available for a reassignment.")
	serveAddress       = flag.String("serve-address", ":8080", "The serving address of task master.")
)

func getTimeoutOrDefault(timeoutStr string) time.Duration {
	if len(timeoutStr) == 0 {
		return *taskDefaultTimeout
	}
	timeSec, err := strconv.Atoi(timeoutStr)
	if err != nil {
		log.Printf("cannot parse `%s` as timeout seconds, using default value instead", timeoutStr)
		return *taskDefaultTimeout
	}
	return time.Duration(timeSec) * time.Second
}

func installHandler() {
	http.HandleFunc("/insert", func(rw http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		taskContent := r.FormValue("task")
		if len(taskContent) == 0 {
			http.Error(rw, "request field `task` cannot be empty", http.StatusBadRequest)
			return
		}
		fmt.Fprint(rw, taskMaster.NewTask(taskContent))
	})
	http.HandleFunc("/query", func(rw http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		task := taskMaster.Query(getTimeoutOrDefault(r.FormValue("timeout_sec")))
		if task == nil {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		encoder := json.NewEncoder(rw)
		encoder.SetIndent("", "    ")
		encoder.Encode((*task))
	})
	http.HandleFunc("/finish", func(rw http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		taskID := r.FormValue("task_id")
		if err := taskMaster.MarkAsComplete(taskID); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
	})
	http.HandleFunc("/status", func(rw http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(rw)
		encoder.SetIndent("", "    ")
		encoder.Encode(taskMaster.GetSnapshot())
	})
}

func main() {
	flag.Parse()
	var err error
	taskMaster, err = task.NewTaskMaster(context.Background(), *snapshotFile, *snapshotInterval)
	if err != nil {
		log.Fatalf("cannot initialize task master: %v", err)
	}
	installHandler()
	log.Printf("task master service is serving on %s", *serveAddress)
	http.ListenAndServe(*serveAddress, nil)
}
