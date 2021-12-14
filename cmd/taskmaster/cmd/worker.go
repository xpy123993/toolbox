package cmd

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/exec"
	"time"

	"github.com/xpy123993/yukino-net/libraries/util"
	"golang.org/x/net/trace"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	pb "github.com/xpy123993/toolbox/proto"
)

func workerRoutinue(
	workerGroup string, timeout time.Duration,
	taskmasterClient pb.TaskMasterClient) error {
	resp, err := taskmasterClient.Query(context.Background(), &pb.QueryRequest{
		Group:        workerGroup,
		LoanDuration: durationpb.New(timeout),
	})
	if err != nil {
		return err
	}
	tracker := trace.New("Worker", fmt.Sprintf("working on task `%s`", resp.GetID()))
	defer tracker.Finish()
	log.Printf("working on task `%s`", resp.GetID())

	command := pb.Command{}
	if err := proto.Unmarshal([]byte(resp.GetData()), &command); err != nil {
		return err
	}
	tracker.LazyPrintf("%s", command.String())

	ctx, cancelFn := context.WithDeadline(context.Background(), resp.Deadline.AsTime().Add(-time.Second))
	defer cancelFn()
	cmd := exec.CommandContext(ctx, command.BaseCommand, command.Arguments...)
	if data, err := cmd.CombinedOutput(); err != nil {
		tracker.LazyPrintf(err.Error())
		tracker.SetError()
		return err
	} else {
		tracker.LazyPrintf("Result: %s", data)
	}
	if _, err := taskmasterClient.Finish(context.Background(), &pb.FinishRequest{
		Group: workerGroup,
		ID:    resp.ID,
	}); err != nil {
		return err
	}
	tracker.LazyPrintf("commited")
	log.Printf("task `%s` is commited", resp.ID)
	return nil
}

func createTaskMasterClient(NetConfig *util.ClientConfig, TaskMasterChannel string) (pb.TaskMasterClient, error) {
	dialer, err := util.CreateClientFromNetConfig(NetConfig)
	if err != nil {
		return nil, err
	}
	client, err := grpc.Dial(TaskMasterChannel, grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return dialer.Dial(s)
	}), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return pb.NewTaskMasterClient(client), nil
}

func worker(NetConfig *util.ClientConfig, TaskMasterChannel string, WorkerGroup string, WorkerTimeout time.Duration) error {
	client, err := createTaskMasterClient(NetConfig, TaskMasterChannel)
	if err != nil {
		return err
	}
	for {
		if err := workerRoutinue(WorkerGroup, WorkerTimeout, client); err != nil {
			log.Printf("worker returns error status: %v", err)
			<-time.After(30 * time.Second)
		}
	}
}

// StartWorker creates a worker job to periodically fetch task from `WorkGroup` of task master.
func StartWorker(NetConfig *util.ClientConfig, TaskMasterChannel string, WorkerGroup string, WorkerTimeout time.Duration) {
	log.Print(worker(NetConfig, TaskMasterChannel, WorkerGroup, WorkerTimeout))
}
