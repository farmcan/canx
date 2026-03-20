package loop

import (
	"strings"

	"github.com/farmcan/canx/internal/tasks"
)

func selectRunnableTasks(items []tasks.Task, maxWorkers int) []int {
	if maxWorkers <= 0 {
		return nil
	}

	selected := make([]int, 0, maxWorkers)
	selectedTasks := make([]tasks.Task, 0, maxWorkers)
	for index, item := range items {
		if item.Status != tasks.StatusPending && item.Status != tasks.StatusInProgress {
			continue
		}
		if len(selected) >= maxWorkers {
			break
		}
		conflict := false
		for _, existing := range selectedTasks {
			if tasksConflict(existing, item) {
				conflict = true
				break
			}
		}
		if conflict {
			continue
		}
		selected = append(selected, index)
		selectedTasks = append(selectedTasks, item)
		if len(item.PlannedFiles) == 0 {
			break
		}
	}
	return selected
}

func canApproveSpawn(parent tasks.Task, items []tasks.Task, request spawnRequest, cfg Config) (bool, string) {
	if parent.SpawnDepth >= cfg.MaxSpawnDepth {
		return false, "max spawn depth reached"
	}
	if childCount(items, parent.ID) >= cfg.MaxChildrenPerTask {
		return false, "max children per task reached"
	}

	candidate := tasks.Task{
		ID:           parent.ID + "/child",
		Goal:         request.Goal,
		Title:        request.Title,
		Status:       tasks.StatusPending,
		PlannedFiles: request.PlannedFiles,
	}
	for _, item := range items {
		if item.ID == parent.ID {
			continue
		}
		if item.Status != tasks.StatusPending && item.Status != tasks.StatusInProgress {
			continue
		}
		if tasksConflict(candidate, item) {
			return false, "planned files conflict with active task"
		}
	}
	return true, ""
}

func childCount(items []tasks.Task, parentID string) int {
	total := 0
	for _, item := range items {
		if item.ParentTaskID == parentID {
			total++
		}
	}
	return total
}

func tasksConflict(a, b tasks.Task) bool {
	if len(a.PlannedFiles) == 0 || len(b.PlannedFiles) == 0 {
		return true
	}
	for _, left := range a.PlannedFiles {
		for _, right := range b.PlannedFiles {
			if strings.TrimSpace(left) != "" && left == right {
				return true
			}
		}
	}
	return false
}
