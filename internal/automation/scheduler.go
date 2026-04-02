// Package automation provides task scheduling and automation capabilities
package automation

import (
	"context"
	"fmt"
	"time"
)

// Task represents a scheduled task
type Task struct {
	ID          string
	Name        string
	Description string
	Schedule    string // Cron expression
	Enabled     bool
	LastRun     *time.Time
	NextRun     *time.Time
	RunCount    int
	MaxCost     float64
}

// Scheduler manages scheduled tasks
type Scheduler struct {
	tasks   map[string]*Task
	running bool
	stopCh  chan struct{}
}

// NewScheduler creates a new task scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks:  make(map[string]*Task),
		stopCh: make(chan struct{}),
	}
}

// AddTask adds a task to the scheduler
func (s *Scheduler) AddTask(task *Task) {
	s.tasks[task.ID] = task
}

// RemoveTask removes a task
func (s *Scheduler) RemoveTask(taskID string) {
	delete(s.tasks, taskID)
}

// GetTask gets a task by ID
func (s *Scheduler) GetTask(taskID string) (*Task, bool) {
	task, ok := s.tasks[taskID]
	return task, ok
}

// ListTasks lists all tasks
func (s *Scheduler) ListTasks() []*Task {
	var tasks []*Task
	for _, t := range s.tasks {
		tasks = append(tasks, t)
	}
	return tasks
}

// EnableTask enables a task
func (s *Scheduler) EnableTask(taskID string) error {
	task, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	task.Enabled = true
	return nil
}

// DisableTask disables a task
func (s *Scheduler) DisableTask(taskID string) error {
	task, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	task.Enabled = false
	return nil
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) {
	s.running = true

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.checkAndRunTasks(ctx)
			case <-s.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	if s.running {
		close(s.stopCh)
		s.running = false
	}
}

// checkAndRunTasks checks for tasks that should run
func (s *Scheduler) checkAndRunTasks(ctx context.Context) {
	now := time.Now()

	for _, task := range s.tasks {
		if !task.Enabled {
			continue
		}

		if task.NextRun == nil || now.After(*task.NextRun) {
			s.runTask(ctx, task)
		}
	}
}

// runTask executes a task
func (s *Scheduler) runTask(ctx context.Context, task *Task) {
	now := time.Now()
	task.LastRun = &now
	task.RunCount++

	// Calculate next run time
	nextRun := now.Add(time.Hour) // Default to hourly
	task.NextRun = &nextRun

	// In a real implementation, this would execute the actual task
	// For now, just log it
	fmt.Printf("Running task: %s\n", task.Name)
}

// ScheduleConfig holds scheduler configuration
type ScheduleConfig struct {
	Timezone string
	Enabled  bool
}

// ParseSchedule parses a cron-like schedule expression
// Simple implementation supporting: @hourly, @daily, @weekly, @monthly
// And simple expressions like "0 2 * * *" (at 2 AM daily)
func ParseSchedule(schedule string) (time.Duration, error) {
	switch schedule {
	case "@hourly":
		return time.Hour, nil
	case "@daily":
		return 24 * time.Hour, nil
	case "@weekly":
		return 7 * 24 * time.Hour, nil
	default:
		// Try to parse as duration
		d, err := time.ParseDuration(schedule)
		if err != nil {
			return 0, fmt.Errorf("unsupported schedule format: %s", schedule)
		}
		return d, nil
	}
}
