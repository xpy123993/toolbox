package taskmaster

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Task describes a task.
type Task struct {
	// ID is the unique identifier of the task.
	ID string `json:"uuid"`
	// Data stores content of the task.
	Data string `json:"data"`
	// AvailableTime specifies the timestamp of the task to be ready.
	AvailableTime time.Time `json:"available_timestamp"`
}

// Snapshot describes a task master snapshot.
type Snapshot struct {
	CreatedAt      time.Time       `json:"creation"`
	AvailableTasks map[string]Task `json:"tasks"`
}

// Scheduler stores all the active tasks.
type Scheduler struct {
	mu         sync.RWMutex
	unsaved    bool
	ownedTasks map[string]Task
}

// Query returns an available task and marked it as assigned.
// This task will be available to assign to other callers after the timeout.
// Returns nil if there is no available task at present.
func (master *Scheduler) Query(timeout time.Duration) *Task {
	master.mu.Lock()
	defer master.mu.Unlock()
	for ID, task := range master.ownedTasks {
		if task.AvailableTime.Before(time.Now()) {
			task.AvailableTime = time.Now().Add(timeout)
			master.ownedTasks[ID] = task
			master.unsaved = true
			return &task
		}
	}
	return nil
}

func (master *Scheduler) ExtendLoan(ID string, deadline time.Time) error {
	master.mu.Lock()
	defer master.mu.Unlock()
	task, ok := master.ownedTasks[ID]
	if !ok {
		return fmt.Errorf("Task `%s` is not found", ID)
	}
	task.AvailableTime = deadline
	master.unsaved = true
	return nil
}

func (master *Scheduler) needsDump() bool {
	master.mu.RLock()
	defer master.mu.RUnlock()
	return master.unsaved
}

// MarkAsComplete marks a task with `ID` as completed state.
// Returns error if task is not found.
func (master *Scheduler) MarkAsComplete(ID string) error {
	master.mu.Lock()
	defer master.mu.Unlock()
	if _, ok := master.ownedTasks[ID]; !ok {
		return fmt.Errorf("Task `%s` is not found", ID)
	}
	delete(master.ownedTasks, ID)
	master.unsaved = true
	return nil
}

// NewTask creates a task that can be assigned immediately.
// Returns the ID in the task master.
func (master *Scheduler) NewTask(Data string) string {
	task := Task{
		ID:            uuid.NewString(),
		Data:          Data,
		AvailableTime: time.Now(),
	}
	master.mu.Lock()
	defer master.mu.Unlock()
	master.ownedTasks[task.ID] = task
	master.unsaved = true
	return task.ID
}

// GetSnapshot returns a snapshot of the task master.
func (master *Scheduler) GetSnapshot() *Snapshot {
	master.mu.RLock()
	defer master.mu.RUnlock()
	return &Snapshot{
		CreatedAt:      time.Now(),
		AvailableTasks: master.ownedTasks,
	}
}

func (master *Scheduler) dumpTo(Filename string) error {
	data, err := json.MarshalIndent(*master.GetSnapshot(), "", "    ")
	if err != nil {
		log.Fatal(err)
	}
	master.mu.Lock()
	master.unsaved = false
	master.mu.Unlock()
	return os.WriteFile(Filename, data, fs.ModePerm)
}

// NewTaskMaster creates a task master which dumps its state to `SnapshotFileName` every `SnapshotInterval`.
func NewTaskMaster(Context context.Context, SnapshotFileName string, SnapshotInterval time.Duration) (*Scheduler, error) {
	taskmaster := Scheduler{
		mu:         sync.RWMutex{},
		unsaved:    true,
		ownedTasks: make(map[string]Task),
	}
	if data, err := os.ReadFile(SnapshotFileName); err == nil {
		snapshot := Snapshot{}
		if err := json.Unmarshal(data, &snapshot); err != nil {
			log.Printf("warning: data corruptted: %v", err)
		}
		taskmaster.unsaved = false
		taskmaster.ownedTasks = snapshot.AvailableTasks
		log.Printf("loaded snapshot created %v ago", time.Since(snapshot.CreatedAt))
	}
	ticker := time.NewTicker(SnapshotInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-Context.Done():
				return
			case <-ticker.C:
				if taskmaster.needsDump() {
					if err := taskmaster.dumpTo(SnapshotFileName); err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	}()
	return &taskmaster, nil
}
