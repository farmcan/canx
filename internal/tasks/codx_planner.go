package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

var ErrPlannerNoTasks = errors.New("planner returned no tasks")

type PlannerRunner interface {
	Run(ctx context.Context, prompt string) (string, error)
}

type CodxPlanner struct {
	Runner PlannerRunner
	PromptBuilder func(goal string) string
}

const plannerPrompt = `You are a software delivery supervisor. Given a goal, output a JSON array of tasks.

Each task must have: id (string), title (string, max 40 chars), goal (string), status ("pending").

Prefer 2-5 tasks when the goal naturally contains multiple steps such as inspect + test + implement.
Use a single task only when the goal is truly atomic.

Output ONLY valid JSON, no explanation. Maximum 5 tasks. Example:
[{"id":"task-1","title":"Add failing test","goal":"write a failing test for X","status":"pending"}]

Goal: `

func DefaultPlannerPrompt(goal string) string {
	return plannerPrompt + goal
}

func (p CodxPlanner) Plan(ctx context.Context, goal string) ([]Task, error) {
	prompt := DefaultPlannerPrompt(goal)
	if p.PromptBuilder != nil {
		prompt = p.PromptBuilder(goal)
	}
	output, err := p.Runner.Run(ctx, prompt)
	if err != nil {
		return nil, err
	}

	items, parseErr := parsePlanJSON(output)
	if parseErr != nil || len(items) == 0 {
		return nil, ErrPlannerNoTasks
	}
	for index := range items {
		if items[index].ID == "" {
			items[index].ID = taskID(goal + "-" + items[index].Title + "-" + items[index].Goal)
		}
		if items[index].Title == "" {
			items[index].Title = titleFromGoal(items[index].Goal)
		}
		items[index].Normalize()
	}
	return items, nil
}

func parsePlanJSON(output string) ([]Task, error) {
	starts := allIndexes(output, "[")
	ends := allIndexes(output, "]")
	for startIndex := len(starts) - 1; startIndex >= 0; startIndex-- {
		start := starts[startIndex]
		for endIndex := len(ends) - 1; endIndex >= 0; endIndex-- {
			end := ends[endIndex]
			if end <= start {
				continue
			}

			var items []Task
			if err := json.Unmarshal([]byte(output[start:end+1]), &items); err == nil {
				return items, nil
			}
		}
	}
	return nil, ErrMissingTaskID
}

func allIndexes(input, needle string) []int {
	indexes := []int{}
	offset := 0
	for {
		index := strings.Index(input[offset:], needle)
		if index == -1 {
			return indexes
		}
		indexes = append(indexes, offset+index)
		offset += index + len(needle)
	}
}
