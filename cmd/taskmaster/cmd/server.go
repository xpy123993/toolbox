package cmd

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/xpy123993/toolbox/pkg/taskmaster"
	"google.golang.org/grpc"

	pb "github.com/xpy123993/toolbox/proto"
)

// StartTaskMasterService creates a task master service on `Channel`.
func StartTaskMasterService(Address string, SnapshotFolder string, SnapshotInterval time.Duration, httpAddr string) {
	flag.Parse()

	listener, err := net.Listen("tcp", Address)
	if err != nil {
		log.Fatal(err)
	}

	taskmaster, err := taskmaster.NewTaskMasterServer(SnapshotFolder, SnapshotInterval)
	if err != nil {
		log.Fatal(err)
	}
	if len(httpAddr) > 0 {
		http.HandleFunc("/tasks", func(rw http.ResponseWriter, r *http.Request) {
			rw.Header().Add("Content-Type", "text/html")
			taskmaster.RenderStatusPage(context.Background(), rw)
		})
		go http.ListenAndServe(httpAddr, nil)
	}
	server := grpc.NewServer()
	server.RegisterService(&pb.TaskMaster_ServiceDesc, taskmaster)
	log.Printf("Serving on %v", listener.Addr())
	server.Serve(listener)
}
