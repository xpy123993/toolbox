package cmd

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"

	"golang.org/x/net/trace"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	pb "github.com/xpy123993/toolbox/proto"
)

const RPCTimeout = 5 * time.Minute

func workerRoutinue(
	backgroundContext context.Context, workerGroup string, timeout time.Duration, taskmasterClient pb.TaskMasterClient) error {

	routineContext, cancelFn := context.WithTimeout(backgroundContext, timeout)
	defer cancelFn()

	resp, err := taskmasterClient.Query(routineContext, &pb.QueryRequest{
		Group:        workerGroup,
		LoanDuration: durationpb.New(RPCTimeout),
	})
	if err != nil {
		return err
	}
	taskID := resp.GetID()

	tracker := trace.New("Worker", fmt.Sprintf("working on task `%s`", taskID))
	defer tracker.Finish()
	log.Printf("working on task `%s`", taskID)

	command := pb.Command{}
	if err := proto.Unmarshal([]byte(resp.GetData()), &command); err != nil {
		return err
	}
	tracker.LazyPrintf("%s", command.String())

	go func() {
		ticker := time.NewTicker(RPCTimeout)
		defer ticker.Stop()
		for {
			select {
			case <-routineContext.Done():
				return
			case <-ticker.C:
				if _, err := taskmasterClient.Extend(routineContext, &pb.TaskExtendRequest{
					Group:        workerGroup,
					ID:           taskID,
					LoanDuration: durationpb.New(2 * RPCTimeout),
				}); err != nil {
					log.Print(err)
					cancelFn()
					return
				}
				tracker.LazyPrintf("RPC deadline refreshed")
			}
		}
	}()

	cmd := exec.CommandContext(routineContext, command.BaseCommand, command.Arguments...)
	data, err := cmd.CombinedOutput()
	if err != nil {
		tracker.LazyPrintf(err.Error())
		tracker.SetError()
		return err
	}
	tracker.LazyPrintf("Result: %s", data)

	if _, err := taskmasterClient.Finish(routineContext, &pb.FinishRequest{
		Group: workerGroup,
		ID:    taskID,
	}); err != nil {
		return err
	}
	tracker.LazyPrintf("commited")
	log.Printf("task `%s` is commited", taskID)
	return nil
}

func createTaskMasterClient(Address string) (pb.TaskMasterClient, error) {
	client, err := grpc.Dial(Address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return pb.NewTaskMasterClient(client), nil
}

func worker(Address string, WorkerGroup string, WorkerTimeout time.Duration) error {
	client, err := createTaskMasterClient(Address)
	if err != nil {
		return err
	}
	for {
		if err := workerRoutinue(context.Background(), WorkerGroup, WorkerTimeout, client); err != nil {
			log.Printf("worker returns error status: %v", err)
			<-time.After(30 * time.Second)
		}
	}
}

// StartWorker creates a worker job to periodically fetch task from `WorkGroup` of task master.
func StartWorker(Address string, WorkerGroup string, WorkerTimeout time.Duration) {
	log.Print(worker(Address, WorkerGroup, WorkerTimeout))
}
