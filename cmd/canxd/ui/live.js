export function computeLiveRefreshPlan(state, run) {
  const tasks = run.tasks || [];
  let nextTaskID = null;
  let refreshTaskDetail = false;

  if (tasks.length > 0) {
    const currentTaskExists = tasks.some((task) => task.id === state.currentTaskID || task.ID === state.currentTaskID);
    const firstTaskID = tasks[0].id || tasks[0].ID || null;
    nextTaskID = currentTaskExists ? state.currentTaskID : firstTaskID;
    refreshTaskDetail = nextTaskID !== null;
  }

  return {
    nextTaskID,
    refreshTaskDetail,
    refreshSession: Boolean(state.currentSessionID) && state.currentSessionID === run.session_id,
  };
}

export function deriveFrontstagePresentation(run) {
  const tasks = run.tasks || [];
  const blockedTask = tasks.find((task) => (task.status || task.Status) === 'blocked');
  if (blockedTask) {
    return {
      phase: 'blocked',
      sceneZone: 'incident_zone',
      actorRole: 'worker',
      displayStatus: `Blocked: ${blockedTask.title || blockedTask.Title || blockedTask.goal || blockedTask.Goal || run.goal || ''}`.trim(),
    };
  }

  const activeTask = tasks.find((task) => (task.status || task.Status) === 'in_progress');
  if (activeTask) {
    return {
      phase: 'working',
      sceneZone: 'workbench',
      actorRole: 'worker',
      displayStatus: `Working: ${activeTask.title || activeTask.Title || activeTask.goal || activeTask.Goal || run.goal || ''}`.trim(),
    };
  }

  const allDone = tasks.length > 0 && tasks.every((task) => (task.status || task.Status) === 'done');
  if (run.status === 'stop' || allDone) {
    return {
      phase: 'done',
      sceneZone: 'sync_port',
      actorRole: 'supervisor',
      displayStatus: `Completed: ${run.reason || run.goal || 'run complete'}`.trim(),
    };
  }

  return {
    phase: 'planning',
    sceneZone: 'command_deck',
    actorRole: 'supervisor',
    displayStatus: `Planning: ${run.goal || 'starting run'}`.trim(),
  };
}
