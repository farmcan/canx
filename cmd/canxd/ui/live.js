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
