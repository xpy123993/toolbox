package taskmaster

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	pb "github.com/xpy123993/toolbox/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ServerImpl is an implementation of task master RPC server.
type ServerImpl struct {
	pb.UnimplementedTaskMasterServer

	mu             sync.RWMutex
	schedulerGroup map[string]*Scheduler

	snapshotFolder   string
	snapshotInterval time.Duration
}

// NewTaskMasterServer creates a ready to use task master server.
func NewTaskMasterServer(SnapshotFolder string, SnapshotInterval time.Duration) (*ServerImpl, error) {
	if err := os.MkdirAll(SnapshotFolder, fs.ModePerm); err != nil {
		return nil, err
	}
	taskMaster := ServerImpl{
		mu:               sync.RWMutex{},
		schedulerGroup:   make(map[string]*Scheduler),
		snapshotFolder:   SnapshotFolder,
		snapshotInterval: SnapshotInterval,
	}
	files, err := filepath.Glob(path.Join(SnapshotFolder, "*.json"))
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		group := strings.TrimSuffix(path.Base(file), ".json")
		taskMaster.schedulerGroup[group], err = NewTaskMaster(context.Background(), file, SnapshotInterval)
		if err != nil {
			return nil, fmt.Errorf("error while loading group `%s`: %v", group, err)
		}
	}
	return &taskMaster, nil
}

// Query implements the RPC method `TaskMaster.Query`.
func (server *ServerImpl) Query(ctx context.Context, request *pb.QueryRequest) (*pb.QueryResponse, error) {
	server.mu.RLock()
	defer server.mu.RUnlock()

	if scheduler, exists := server.schedulerGroup[request.GetGroup()]; exists && scheduler != nil {
		task := scheduler.Query(request.LoanDuration.AsDuration())
		if task == nil {
			return nil, status.Errorf(codes.NotFound, "no available tasks at present")
		}
		return &pb.QueryResponse{
			ID:       task.ID,
			Data:     task.Data,
			Deadline: timestamppb.New(task.AvailableTime),
		}, nil
	}
	return nil, status.Errorf(codes.NotFound, "group not found")
}

// Finish implements the RPC method `TaskMaster.Finish`.
func (server *ServerImpl) Finish(ctx context.Context, request *pb.FinishRequest) (*pb.FinishResponse, error) {
	server.mu.RLock()
	defer server.mu.RUnlock()

	if scheduler, exists := server.schedulerGroup[request.GetGroup()]; exists && scheduler != nil {
		err := scheduler.MarkAsComplete(request.GetID())
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "no active task with ID `%s`", request.GetID())
		}
		return &pb.FinishResponse{}, nil
	}
	return nil, status.Errorf(codes.NotFound, "group not found")
}

// Insert implements the RPC method `TaskMaster.Insert`.
func (server *ServerImpl) Insert(ctx context.Context, request *pb.InsertRequest) (*pb.InsertResponse, error) {
	server.mu.Lock()
	scheduler, exists := server.schedulerGroup[request.GetGroup()]
	if !exists {
		var err error
		scheduler, err = NewTaskMaster(context.Background(), path.Join(server.snapshotFolder, fmt.Sprintf("%s.json", request.GetGroup())), server.snapshotInterval)
		if err != nil {
			server.mu.Unlock()
			return nil, status.Errorf(codes.Internal, "error returned while the initialization: %v", err.Error())
		}
		server.schedulerGroup[request.GetGroup()] = scheduler
	}
	server.mu.Unlock()
	return &pb.InsertResponse{
		ID: scheduler.NewTask(request.Data),
	}, nil
}

// RenderStatusPage renders a basic HTML page to show the contents inside the server.
func (server *ServerImpl) RenderStatusPage(ctx context.Context, writer io.Writer) {
	server.mu.RLock()
	defer server.mu.RUnlock()
	fmt.Fprintf(writer, "<h2> Total group count: %d </h2>\n", len(server.schedulerGroup))
	for groupName, scheduler := range server.schedulerGroup {
		snapshot := scheduler.GetSnapshot()
		fmt.Fprintf(writer, "<div>\n")
		fmt.Fprintf(writer, "<h3> Group `%s` </h3>\n", groupName)
		fmt.Fprintf(writer, "<h4> Task Number: %d </h4>\n", len(snapshot.AvailableTasks))
		for ID, task := range snapshot.AvailableTasks {
			label := "Pending"
			if task.AvailableTime.After(time.Now()) {
				label = "Working"
			}
			fmt.Fprintf(writer, "<div><b>[%s]</b> %s</div>\n", label, ID)
		}
		fmt.Fprintf(writer, "</div>\n")
	}
}
