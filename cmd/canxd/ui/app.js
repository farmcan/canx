let currentRun = null;
let currentContext = null;
let currentRoom = null;
let currentTask = null;
let currentEventSource = null;

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
  const [run, events, actions, context, sessions, rooms] = await Promise.all([
    fetchJSON(`/api/runs/${runID}`),
    fetchJSON(`/api/runs/${runID}/events`),
    fetchJSON(`/api/runs/${runID}/actions`),
    fetchJSON('/api/context'),
    fetchJSON('/api/sessions'),
    fetchJSON('/api/rooms')
  ]);

  currentRun = run;
  currentContext = context;

  setText('run-summary', `${run.goal}\n${run.reason || ''}`);
  setText('session-summary', run.session_id || '—');
  setText('task-summary', `${run.task_count} tasks`);
  document.getElementById('events').textContent = JSON.stringify(events, null, 2);
  document.getElementById('actions').textContent = JSON.stringify(actions, null, 2);

  renderTasks(run);
  renderSessions(sessions, run.session_id);
  await renderSession(run.session_id);
  renderContext(context);
  renderRooms(rooms, run.id);
  startEventStream(run.id);
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
      currentTask = task.id || task.ID;
      const detail = await fetchJSON(`/api/runs/${run.id}/tasks/${task.id || task.ID}`);
      document.getElementById('task-detail').textContent = JSON.stringify(detail, null, 2);
    };
    container.appendChild(row);
  });

  if ((run.tasks || []).length > 0) {
    currentTask = run.tasks[0].id || run.tasks[0].ID;
    document.getElementById('task-detail').textContent = JSON.stringify(run.tasks[0], null, 2);
  }
}

async function renderSession(sessionID) {
  if (!sessionID) {
    document.getElementById('session-detail').textContent = 'No session';
    return;
  }
  const report = await fetchJSON(`/api/sessions/${sessionID}`);
  const session = report.session || report.Session;
  const runtime = report.runtime || report.Runtime || {};
  const turns = session.turns || session.Turns || [];
  const container = document.getElementById('session-detail');
  container.innerHTML = `
    <div class="detail-card"><strong>ID</strong><div>${session.id || session.ID}</div></div>
    <div class="detail-card"><strong>Label</strong><div>${session.label || session.Label || ''}</div></div>
    <div class="detail-card"><strong>Runtime</strong><div>model=${runtime.model || runtime.Model || ''} sandbox=${runtime.sandbox || runtime.Sandbox || ''} approval=${runtime.approval || runtime.Approval || ''}</div></div>
    <div class="detail-card"><strong>Summary</strong><pre>${session.lastSummary || session.LastSummary || ''}</pre></div>
    <div class="detail-card"><strong>Turns</strong><pre>${JSON.stringify(turns, null, 2)}</pre></div>
  `;
}

function renderSessions(reports, selectedID) {
  const container = document.getElementById('sessions');
  container.innerHTML = '';
  reports.forEach((report) => {
    const session = report.session || report.Session;
    const row = document.createElement('button');
    row.className = 'task-row';
    row.innerHTML = `
      <div class="task-row-head">
        <strong>${session.id || session.ID}</strong>
        ${statusBadge(report.decision || report.Decision)}
      </div>
      <div class="run-reason">${session.label || session.Label || ''}</div>
    `;
    row.onclick = () => renderSession(session.id || session.ID);
    if ((session.id || session.ID) === selectedID) {
      row.classList.add('selected');
    }
    container.appendChild(row);
  });
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
  const highlights = document.getElementById('context-highlights');
  highlights.innerHTML = `
    <div class="detail-card"><strong>Latest spec</strong><div>${context.latest_spec_path || '—'}</div></div>
    <div class="detail-card"><strong>Latest plan</strong><div>${context.latest_plan_path || '—'}</div></div>
    <div class="detail-card"><strong>AGENTS visible</strong><div>${context.agents ? 'yes' : 'no'}</div></div>
  `;

  const docsList = document.getElementById('docs-list');
  docsList.innerHTML = '';
  (context.docs || []).forEach((doc) => {
    const button = document.createElement('button');
    button.className = 'task-row compact';
    button.textContent = doc.path || doc.Path;
    button.onclick = async () => {
      const detail = await fetchJSON(`/api/context/docs/${doc.path || doc.Path}`);
      setText('context-detail', detail.content || '');
    };
    docsList.appendChild(button);
  });
}

function renderRooms(rooms, runID) {
  const container = document.getElementById('rooms');
  container.innerHTML = '';
  const filtered = rooms.filter((room) => !runID || room.run_id === runID || room.run_id === '');
  filtered.forEach((room) => {
    const button = document.createElement('button');
    button.className = 'task-row';
    button.innerHTML = `
      <div class="task-row-head">
        <strong>${room.title}</strong>
        <span class="run-meta">${room.id}</span>
      </div>
      <div class="run-reason">${room.run_id || ''}</div>
    `;
    button.onclick = () => loadRoom(room.id);
    container.appendChild(button);
  });
  if (filtered.length > 0) {
    loadRoom(filtered[0].id);
  } else {
    document.getElementById('room-messages').textContent = 'No rooms yet.';
    currentRoom = null;
  }
}

async function loadRoom(roomID) {
  currentRoom = roomID;
  const messages = await fetchJSON(`/api/rooms/${roomID}/messages`);
  document.getElementById('room-messages').textContent = JSON.stringify(messages, null, 2);
}

function startEventStream(runID) {
  if (currentEventSource) {
    currentEventSource.close();
  }
  currentEventSource = new EventSource(`/api/runs/${runID}/events/stream`);
  currentEventSource.onmessage = async () => {
    const [events, actions] = await Promise.all([
      fetchJSON(`/api/runs/${runID}/events`),
      fetchJSON(`/api/runs/${runID}/actions`)
    ]);
    document.getElementById('events').textContent = JSON.stringify(events, null, 2);
    document.getElementById('actions').textContent = JSON.stringify(actions, null, 2);
  };
  currentEventSource.onerror = () => {
    currentEventSource.close();
  };
}

document.getElementById('refresh').onclick = () => {
  currentRun = null;
  loadRuns().catch((error) => {
    setText('events', String(error));
  });
};

document.getElementById('room-form').onsubmit = async (event) => {
  event.preventDefault();
  if (!currentRoom) {
    return;
  }
  const input = document.getElementById('room-message');
  const body = input.value.trim();
  if (!body) {
    return;
  }
  await fetch(`/api/rooms/${currentRoom}/messages`, {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
      participant_id: 'human-local',
      role: 'human',
      kind: 'instruction',
      task_id: currentTask || '',
      body
    })
  });
  input.value = '';
  await loadRoom(currentRoom);
};

loadRuns().catch((error) => {
  setText('events', String(error));
});
