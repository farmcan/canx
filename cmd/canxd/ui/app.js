let currentRun = null;
let currentContext = null;

async function fetchJSON(path) {
  const response = await fetch(path);
  if (!response.ok) {
    throw new Error(`${path}: ${response.status}`);
  }
  return response.json();
}

function statusBadge(status) {
  return `<span class="badge badge-${status || 'unknown'}">${status || 'unknown'}</span>`;
}

function setText(id, value) {
  document.getElementById(id).textContent = value;
}

async function loadRuns() {
  const runs = await fetchJSON('/api/runs');
  const container = document.getElementById('runs');
  container.innerHTML = '';

  runs.forEach((run) => {
    const card = document.createElement('button');
    card.className = 'run-card';
    card.innerHTML = `
      <div class="run-head">
        <strong>${run.id}</strong>
        ${statusBadge(run.status)}
      </div>
      <div class="run-goal">${run.goal}</div>
      <div class="run-meta">turns=${run.turn_count} · tasks=${run.task_count}</div>
      <div class="run-reason">${run.reason || ''}</div>
    `;
    card.onclick = () => loadRun(run.id);
    container.appendChild(card);
  });

  if (!currentRun && runs.length > 0) {
    await loadRun(runs[0].id);
  }
}

async function loadRun(runID) {
  const [run, events, context] = await Promise.all([
    fetchJSON(`/api/runs/${runID}`),
    fetchJSON(`/api/runs/${runID}/events`),
    fetchJSON('/api/context')
  ]);

  currentRun = run;
  currentContext = context;

  setText('run-summary', `${run.goal}\n${run.reason || ''}`);
  setText('session-summary', run.session_id || '—');
  setText('task-summary', `${run.task_count} tasks`);
  document.getElementById('events').textContent = JSON.stringify(events, null, 2);

  renderTasks(run);
  await renderSession(run.session_id);
  renderContext(context);
}

function renderTasks(run) {
  const container = document.getElementById('tasks');
  container.innerHTML = '';

  (run.tasks || []).forEach((task) => {
    const row = document.createElement('button');
    row.className = 'task-row';
    row.innerHTML = `
      <div class="task-row-head">
        <strong>${task.title || task.ID}</strong>
        ${statusBadge(task.status || task.Status)}
      </div>
      <div class="task-row-goal">${task.goal || task.Goal}</div>
    `;
    row.onclick = async () => {
      const detail = await fetchJSON(`/api/runs/${run.id}/tasks/${task.id || task.ID}`);
      document.getElementById('task-detail').textContent = JSON.stringify(detail, null, 2);
    };
    container.appendChild(row);
  });

  if ((run.tasks || []).length > 0) {
    document.getElementById('task-detail').textContent = JSON.stringify(run.tasks[0], null, 2);
  }
}

async function renderSession(sessionID) {
  if (!sessionID) {
    setText('session-detail', 'No session');
    return;
  }
  const report = await fetchJSON(`/api/sessions/${sessionID}`);
  document.getElementById('session-detail').textContent = JSON.stringify(report, null, 2);
}

function renderContext(context) {
  const tabs = document.getElementById('context-tabs');
  tabs.innerHTML = '';

  const items = [
    {label: 'README', value: context.readme || ''},
    {label: 'AGENTS', value: context.agents || ''},
    {label: 'Docs', value: JSON.stringify(context.docs || [], null, 2)}
  ];

  items.forEach((item, index) => {
    const button = document.createElement('button');
    button.className = `tab ${index === 0 ? 'active' : ''}`;
    button.textContent = item.label;
    button.onclick = () => {
      document.querySelectorAll('.tab').forEach((node) => node.classList.remove('active'));
      button.classList.add('active');
      setText('context-detail', item.value);
    };
    tabs.appendChild(button);
  });

  setText('context-detail', items[0].value);
}

document.getElementById('refresh').onclick = () => {
  currentRun = null;
  loadRuns().catch((error) => {
    setText('events', String(error));
  });
};

loadRuns().catch((error) => {
  setText('events', String(error));
});
