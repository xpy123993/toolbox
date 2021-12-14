package cmd

import (
	"context"
	"fmt"

	"github.com/xpy123993/yukino-net/libraries/util"
	"google.golang.org/protobuf/proto"

	pb "github.com/xpy123993/toolbox/proto"
)

// InsertTask inserts a task into `WorkerGroup` of the task master.
func InsertTask(Context context.Context, NetConfig *util.ClientConfig, TaskMasterChannel string, WorkerGroup string, BaseCommand string, Arguments []string) error {
	client, err := createTaskMasterClient(NetConfig, TaskMasterChannel)
	if err != nil {
		return err
	}
	data, err := proto.Marshal(&pb.Command{BaseCommand: BaseCommand, Arguments: Arguments})
	if err != nil {
		return err
	}

	resp, err := client.Insert(Context, &pb.InsertRequest{
		Group: WorkerGroup,
		Data:  string(data),
	})
	if err != nil {
		return err
	}
	fmt.Printf("Task is successfully committed with ID `%s`.\n", resp.GetID())
	return nil
}
