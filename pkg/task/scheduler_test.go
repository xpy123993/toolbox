package task_test

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/xpy123993/toolbox/pkg/task"
)

func TestQuery(t *testing.T) {
	taskMaster, err := task.NewTaskMaster(context.Background(), path.Join(t.TempDir(), "test.json"), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	taskMaster.NewTask("test")
	task := taskMaster.Query(5 * time.Millisecond)
	if task == nil {
		t.Fatal("expect a task to be returned")
	}
	if task = taskMaster.Query(time.Millisecond); task != nil {
		t.Error("expect nothing to be returned")
	}
	time.Sleep(5 * time.Millisecond)
	if task = taskMaster.Query(time.Millisecond); task == nil {
		t.Error("expect a task to be returned")
	}
}

func TestDumpSnapshot(t *testing.T) {
	snapshotFile := path.Join(t.TempDir(), "test.json")
	taskMaster, err := task.NewTaskMaster(context.Background(), snapshotFile, 2*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	taskMaster.NewTask("test")
	time.Sleep(3 * time.Millisecond)
	taskMaster.GetSnapshot()
	data, err := os.ReadFile(snapshotFile)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := task.Snapshot{}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.AvailableTasks) != 1 {
		t.Fail()
	}
	for _, value := range snapshot.AvailableTasks {
		if value.Data != "test" {
			t.Fail()
		}
	}
}

func TestRecoverFromSnapshot(t *testing.T) {
	snapshotFile := path.Join(t.TempDir(), "test.json")
	taskUUID := uuid.NewString()
	snapshot := task.Snapshot{
		CreatedAt: time.Now(),
		AvailableTasks: map[string]task.Task{
			taskUUID: {
				ID:            taskUUID,
				Data:          "test",
				AvailableTime: time.Now(),
			},
		},
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(snapshotFile, data, 0644); err != nil {
		t.Fatal(err)
	}
	taskMaster, err := task.NewTaskMaster(context.Background(), snapshotFile, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	task := taskMaster.Query(time.Minute)
	if task == nil {
		t.Fatal("expect a task to be returned")
	}
	if task.Data != "test" {
		t.Errorf("unexpect data: %s", task.Data)
	}
}

func TestMarkAsComplete(t *testing.T) {
	taskMaster, err := task.NewTaskMaster(context.Background(), path.Join(t.TempDir(), "test.json"), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	taskID := taskMaster.NewTask("test")
	if len(taskMaster.GetSnapshot().AvailableTasks) == 0 {
		t.Fail()
	}
	if err := taskMaster.MarkAsComplete(taskID); err != nil {
		t.Error(err)
	}
	if len(taskMaster.GetSnapshot().AvailableTasks) != 0 {
		t.Fail()
	}
}
