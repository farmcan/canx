import {computeLiveRefreshPlan, deriveFrontstagePresentation} from './live.js';

let currentRun = null;
let currentContext = null;
let currentRoom = null;
let currentTask = null;
let currentEventSource = null;
let currentMode = 'backstage';
let currentPresentation = null;

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
  const presentation = await fetchJSON(`/api/runs/${runID}/presentation`).catch(() => deriveFrontstagePresentation(run));

  currentRun = run;
  currentContext = context;
  currentPresentation = normalizePresentation(presentation, run);

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
  renderFrontstage(run, currentPresentation);
  startEventStream(run.id);
}

async function renderTaskDetail(runID, taskID) {
  if (!taskID) {
    document.getElementById('task-detail').textContent = 'Select a task.';
    return;
  }
  const detail = await fetchJSON(`/api/runs/${runID}/tasks/${taskID}`);
  document.getElementById('task-detail').textContent = JSON.stringify(detail, null, 2);
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
  const turns = report.turns || report.Turns || session.turns || session.Turns || [];
  const container = document.getElementById('session-detail');
  container.innerHTML = `
    <div class="detail-card"><strong>ID</strong><div>${session.id || session.ID}</div></div>
    <div class="detail-card"><strong>Label</strong><div>${session.label || session.Label || ''}</div></div>
    <div class="detail-card"><strong>Runtime</strong><div>model=${runtime.model || runtime.Model || ''} sandbox=${runtime.sandbox || runtime.Sandbox || ''} approval=${runtime.approval || runtime.Approval || ''}</div></div>
    <div class="detail-card"><strong>Summary</strong><pre>${session.lastSummary || session.LastSummary || ''}</pre></div>
  `;
  const turnsCard = document.createElement('div');
  turnsCard.className = 'detail-card';
  const title = document.createElement('strong');
  title.textContent = 'Turns';
  turnsCard.appendChild(title);
  if (turns.length === 0) {
    const empty = document.createElement('div');
    empty.textContent = 'No turns';
    turnsCard.appendChild(empty);
  } else {
    turns.forEach((turn, index) => {
      const details = document.createElement('details');
      details.className = 'turn-details';
      const summary = document.createElement('summary');
      summary.textContent = `Turn ${index + 1}`;
      details.appendChild(summary);
      const pre = document.createElement('pre');
      pre.textContent = typeof turn === 'string' ? turn : JSON.stringify(turn, null, 2);
      details.appendChild(pre);
      turnsCard.appendChild(details);
    });
  }
  container.appendChild(turnsCard);
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
    {label: 'Docs', value: 'Select a file from the tree below.'}
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
  renderDocsTree(context.docs || [], docsList);
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

function normalizePresentation(presentation, run) {
  if (!presentation) {
    return deriveFrontstagePresentation(run);
  }
  return {
    phase: presentation.phase || presentation.Phase || 'planning',
    sceneZone: presentation.scene_zone || presentation.SceneZone || 'command_deck',
    displayStatus: presentation.display_status || presentation.DisplayStatus || '',
    actorRole: presentation.actor_role || presentation.ActorRole || 'supervisor',
  };
}

function renderFrontstage(run, presentation) {
  const taskTitle = (run.tasks || []).find((task) => (task.status || task.Status) === 'in_progress' || (task.status || task.Status) === 'blocked')?.title
    || (run.tasks || []).find((task) => (task.status || task.Status) === 'in_progress' || (task.status || task.Status) === 'blocked')?.Title
    || (run.tasks || [])[0]?.title
    || (run.tasks || [])[0]?.Title
    || 'No active task';
  setText('frontstage-run', run.goal || 'Select a run');
  setText('frontstage-phase', presentation.phase || '—');
  setText('frontstage-task', taskTitle);
  setText('frontstage-status', presentation.displayStatus || 'Select a run.');
  setText('frontstage-strip', `status=${run.status || 'unknown'} · phase=${presentation.phase || 'planning'} · session=${run.session_id || '—'}`);

  document.querySelectorAll('.scene-zone').forEach((zone) => {
    zone.classList.toggle('active', zone.dataset.zone === presentation.sceneZone);
  });

  const avatar = document.getElementById('scene-avatar');
  avatar.className = `scene-avatar phase-${presentation.phase || 'planning'}`;
  const positions = {
    command_deck: {top: '86px', left: '120px'},
    workbench: {top: '118px', left: '70%'},
    test_lab: {top: '300px', left: '112px'},
    review_gate: {top: '304px', left: '47%'},
    sync_port: {top: '390px', left: '74%'},
    incident_zone: {top: '402px', left: '132px'},
  };
  const position = positions[presentation.sceneZone] || positions.command_deck;
  avatar.style.top = position.top;
  avatar.style.left = position.left;
  setText('avatar-bubble', presentation.displayStatus || 'Waiting for run data.');
}

function renderDocsTree(docs, container) {
  container.innerHTML = '';
  const tree = {};
  docs.forEach((doc) => {
    const path = doc.path || doc.Path;
    const parts = path.split('/');
    let node = tree;
    parts.forEach((part, index) => {
      if (!node[part]) {
        node[part] = index === parts.length - 1 ? {__file: path} : {};
      }
      node = node[part];
    });
  });
  container.appendChild(renderTreeNode(tree));
}

function renderTreeNode(node) {
  const wrapper = document.createElement('div');
  wrapper.className = 'tree';
  Object.keys(node).sort().forEach((key) => {
    const value = node[key];
    if (value.__file) {
      const file = document.createElement('button');
      file.className = 'task-row compact tree-file';
      file.textContent = key;
      file.onclick = async () => {
        const detail = await fetchJSON(`/api/context/docs/${value.__file}`);
        setText('context-detail', detail.content || '');
      };
      wrapper.appendChild(file);
      return;
    }
    const folder = document.createElement('details');
    folder.className = 'tree-folder';
    if (key === 'docs') {
      folder.open = true;
    }
    const summary = document.createElement('summary');
    summary.textContent = key;
    folder.appendChild(summary);
    folder.appendChild(renderTreeNode(value));
    wrapper.appendChild(folder);
  });
  return wrapper;
}

function startEventStream(runID) {
  if (currentEventSource) {
    currentEventSource.close();
  }
  currentEventSource = new EventSource(`/api/runs/${runID}/events/stream`);
  currentEventSource.onmessage = async () => {
    const [run, events, actions, sessions] = await Promise.all([
      fetchJSON(`/api/runs/${runID}`),
      fetchJSON(`/api/runs/${runID}/events`),
      fetchJSON(`/api/runs/${runID}/actions`),
      fetchJSON('/api/sessions')
    ]);
    const presentation = await fetchJSON(`/api/runs/${runID}/presentation`).catch(() => deriveFrontstagePresentation(run));
    currentRun = run;
    currentPresentation = normalizePresentation(presentation, run);
    setText('run-summary', `${run.goal}\n${run.reason || ''}`);
    setText('session-summary', run.session_id || '—');
    setText('task-summary', `${run.task_count} tasks`);
    renderTasks(run);
    renderSessions(sessions, run.session_id);

    const plan = computeLiveRefreshPlan({
      currentTaskID: currentTask,
      currentSessionID: run.session_id,
    }, run);
    currentTask = plan.nextTaskID;
    if (plan.refreshTaskDetail) {
      await renderTaskDetail(run.id, currentTask);
    }
    if (plan.refreshSession) {
      await renderSession(run.session_id);
    }
    renderFrontstage(run, currentPresentation);
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

function setMode(mode) {
  currentMode = mode;
  document.getElementById('frontstage').classList.toggle('hidden', mode !== 'frontstage');
  document.getElementById('backstage').classList.toggle('hidden', mode !== 'backstage');
  document.getElementById('mode-frontstage').classList.toggle('active', mode === 'frontstage');
  document.getElementById('mode-backstage').classList.toggle('active', mode === 'backstage');
}

document.getElementById('mode-frontstage').onclick = () => setMode('frontstage');
document.getElementById('mode-backstage').onclick = () => setMode('backstage');

loadRuns().catch((error) => {
  setText('events', String(error));
});
