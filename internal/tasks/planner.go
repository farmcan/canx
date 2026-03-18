package tasks

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"strings"
)

type Planner interface {
	Plan(ctx context.Context, goal string) ([]Task, error)
}

type SingleTaskPlanner struct{}

func (SingleTaskPlanner) Plan(_ context.Context, goal string) ([]Task, error) {
	task := Task{
		ID:     taskID(goal),
		Title:  titleFromGoal(goal),
		Goal:   goal,
		Status: StatusPending,
	}
	task.Normalize()
	return []Task{task}, nil
}

func titleFromGoal(goal string) string {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return "Task"
	}
	if len(goal) <= 40 {
		return goal
	}
	return goal[:40]
}

func taskID(goal string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(goal)))
	return "task-" + hex.EncodeToString(sum[:4])
}
