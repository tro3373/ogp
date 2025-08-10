package cmd

import (
	"context"
	"fmt"
	"sync"
)

type Task struct {
	No          int
	Ctx         context.Context
	Wg          *sync.WaitGroup
	Queue       chan string
	ResultQueue chan *TaskResult
}

func NewTask(no int, ctx context.Context, wg *sync.WaitGroup, queue chan string, resultQueue chan *TaskResult) *Task {
	wg.Add(1)
	return &Task{
		No:          no,
		Ctx:         ctx,
		Wg:          wg,
		Queue:       queue,
		ResultQueue: resultQueue,
	}
}

func (tr *Task) String() string {
	return fmt.Sprintf("Task: { No:%d }", tr.No)
}
