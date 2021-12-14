package cmd

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/xpy123993/toolbox/pkg/taskmaster"
	"github.com/xpy123993/yukino-net/libraries/util"
	"google.golang.org/grpc"

	pb "github.com/xpy123993/toolbox/proto"
)

// StartTaskMasterService creates a task master service on `Channel`.
func StartTaskMasterService(NetConfig *util.ClientConfig, Channel string, SnapshotFolder string, SnapshotInterval time.Duration) {
	flag.Parse()

	listener, err := util.CreateListenerFromNetConfig(NetConfig, Channel)
	if err != nil {
		log.Fatal(err)
	}

	taskmaster, err := taskmaster.NewTaskMasterServer(SnapshotFolder, SnapshotInterval)
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/tasks", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("Content-Type", "text/html")
		taskmaster.RenderStatusPage(context.Background(), rw)
	})
	server := grpc.NewServer()
	server.RegisterService(&pb.TaskMaster_ServiceDesc, taskmaster)
	server.Serve(listener)
}
