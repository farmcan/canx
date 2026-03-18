package tasks

import "strings"

type Planner interface {
	Plan(goal string) ([]Task, error)
}

type StaticPlanner struct{}

func (StaticPlanner) Plan(goal string) ([]Task, error) {
	task := Task{
		ID:     "task-1",
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
