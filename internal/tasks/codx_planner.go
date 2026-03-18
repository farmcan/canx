package tasks

import (
	"context"
	"encoding/json"
	"strings"
)

type PlannerRunner interface {
	Run(ctx context.Context, prompt string) (string, error)
}

type CodxPlanner struct {
	Runner PlannerRunner
}

const plannerPrompt = `You are a software delivery supervisor. Given a goal, output a JSON array of tasks.

Each task must have: id (string), title (string, max 40 chars), goal (string), status ("pending").

Output ONLY valid JSON, no explanation. Maximum 5 tasks. Example:
[{"id":"task-1","title":"Add failing test","goal":"write a failing test for X","status":"pending"}]

Goal: `

func (p CodxPlanner) Plan(ctx context.Context, goal string) ([]Task, error) {
	output, err := p.Runner.Run(ctx, plannerPrompt+goal)
	if err != nil {
		return nil, err
	}

	items, err := parsePlanJSON(output)
	if err != nil || len(items) == 0 {
		return SingleTaskPlanner{}.Plan(ctx, goal)
	}
	for index := range items {
		items[index].Normalize()
	}
	return items, nil
}

func parsePlanJSON(output string) ([]Task, error) {
	start := strings.Index(output, "[")
	end := strings.LastIndex(output, "]")
	if start == -1 || end == -1 || end <= start {
		return nil, ErrMissingTaskID
	}

	var items []Task
	if err := json.Unmarshal([]byte(output[start:end+1]), &items); err != nil {
		return nil, err
	}
	return items, nil
}
