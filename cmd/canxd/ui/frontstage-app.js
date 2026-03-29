import {
  DEFAULT_DEMO_BEATS,
  nextDemoState,
  buildInteractionFeed,
  beatFromPresentation,
  detectFrontstageUpdate,
  personaSpeechFromSummary,
} from './frontstage-core.js';
import {assertFrontstageDriver, createFrontstageController} from './frontstage-driver.js';
import {createPhaserFrontstageDriver} from './phaser-frontstage-driver.js';

const ZONES = {
  command_deck: {x: 170, y: 130, width: 240, height: 130, label: 'Command Deck'},
  workbench: {x: 560, y: 120, width: 260, height: 170, label: 'Workbench'},
  test_lab: {x: 140, y: 340, width: 220, height: 140, label: 'Test Lab'},
  review_gate: {x: 400, y: 330, width: 220, height: 130, label: 'Review Gate'},
  sync_port: {x: 690, y: 390, width: 180, height: 120, label: 'Sync Port'},
  incident_zone: {x: 80, y: 520, width: 220, height: 90, label: 'Incident'},
};

const interactionSeed = [
  {role: 'system', body: 'Run queued for frontstage playback.'},
  {role: 'human', body: 'Make the workbench animation feel more confident.'},
  {role: 'agent', body: 'Validation window reserved for future live effects.'},
  {role: 'agent', body: 'Realtime interaction window ready for event bindings.'},
];

const state = {
  isPlaying: false,
  currentBeat: null,
  currentZone: 'command_deck',
  tick: 0,
  pulsingLive: false,
  liveRunID: null,
  liveBeats: [],
  autoPlayTimer: null,
  refreshTimer: null,
  lastSeenBeatCount: 0,
};

const refs = {
  runName: document.getElementById('run-name'),
  phaseName: document.getElementById('phase-name'),
  taskName: document.getElementById('task-name'),
  statusLine: document.getElementById('status-line'),
  phaseStrip: document.getElementById('phase-strip'),
  interactionList: document.getElementById('interaction-list'),
};

renderPhaseStrip();
renderInteractionList();

let sceneRef = null;
const driver = assertFrontstageDriver(createPhaserFrontstageDriver({
  Phaser,
  containerId: 'phaser-stage',
  zones: ZONES,
}));
const controller = createFrontstageController(driver, {
  beats: DEFAULT_DEMO_BEATS,
});
controller.mount();

function renderCurrentState() {
  const current = DEFAULT_DEMO_BEATS.find((beat) => beat.id === state.currentBeat) || {
    id: 'idle',
    label: 'Idle',
    summary: 'Waiting for start',
    zone: 'command_deck',
    color: 0x60a5fa,
  };

  refs.phaseName.textContent = current.label;
  refs.taskName.textContent = current.summary;
  refs.statusLine.textContent = personaSpeechFromSummary(current.summary);
  refs.phaseStrip.querySelectorAll('.phase-pill').forEach((node) => {
    node.classList.toggle('active', node.dataset.phase === current.id);
  });
  renderInteractionList(current);
  controller.renderBeat({
    ...current,
    summary: personaSpeechFromSummary(current.summary),
  });
  controller.setInteractionFeed(buildInteractionFeed([
    ...interactionSeed,
    {role: 'beat', body: current.summary},
  ], 4));
}

function renderPhaseStrip() {
  refs.phaseStrip.innerHTML = '';
  DEFAULT_DEMO_BEATS.forEach((phase) => {
    const pill = document.createElement('div');
    pill.className = 'phase-pill';
    pill.dataset.phase = phase.id;
    pill.textContent = phase.label;
    refs.phaseStrip.appendChild(pill);
  });
}

function renderInteractionList(current = {summary: 'Awaiting start'}) {
  refs.interactionList.innerHTML = '';
  const feed = buildInteractionFeed([
    ...interactionSeed,
    {role: 'beat', body: current.summary},
  ], 4);
  feed.forEach((item) => {
    const wrapper = document.createElement('div');
    wrapper.className = 'interaction-item';
    wrapper.innerHTML = `<div class="meta">${item.role}</div><div>${item.body}</div>`;
    refs.interactionList.appendChild(wrapper);
  });
}

document.getElementById('start-demo').onclick = () => {
  Object.assign(state, nextDemoState(state));
  renderCurrentState();
};

document.getElementById('next-phase').onclick = () => {
  Object.assign(state, nextDemoState({...state, isPlaying: true}));
  renderCurrentState();
};

document.getElementById('toggle-live').onclick = () => {
  state.pulsingLive = !state.pulsingLive;
  refs.statusLine.textContent = state.pulsingLive
    ? 'Realtime interaction pulse enabled. Future events can trigger review-gate effects here.'
    : 'Realtime interaction pulse disabled.';
  renderCurrentState();
};

document.getElementById('load-latest-run').onclick = async () => {
  await loadLatestRun();
};

document.getElementById('auto-play-run').onclick = async () => {
  await loadLatestRun();
  startAutoPlay();
};

window.frontstageBeatFromPresentation = beatFromPresentation;

async function loadLatestRun() {
  const payload = await fetchJSON('/api/frontstage/latest');
  if (!payload.run || !payload.run.id) {
    refs.statusLine.textContent = 'No runs available.';
    return;
  }
  const run = payload.run;
  const presentation = payload.presentation || {};
  const beats = payload.beats || [];
  const timeline = payload.timeline || beats;
  const actions = payload.actions || [];
  const messages = payload.messages || [];

  state.liveRunID = run.id;
  state.liveBeats = timeline;
  state.lastSeenBeatCount = timeline.length;
  refs.runName.textContent = run.goal || run.id;
  refs.statusLine.textContent = presentation.display_status || 'Loaded latest run.';
  refs.taskName.textContent = presentation.display_status || 'Loaded latest run.';
  renderInteractionList({
    summary: presentation.display_status || 'Loaded latest run.',
  }, timeline);

  const firstBeat = timeline[0];
  if (firstBeat) {
    state.currentBeat = firstBeat.type;
    renderBeatByType(firstBeat.type, firstBeat.summary);
  }
  if (timeline.length > 1 && !state.autoPlayTimer) {
    startAutoPlay();
  }
}

function renderBeatByType(beatType, summaryOverride = '') {
  const current = DEFAULT_DEMO_BEATS.find((beat) => beat.id === beatType) || DEFAULT_DEMO_BEATS[0];
  const effectiveSummary = summaryOverride || current.summary;
  refs.phaseName.textContent = current.label;
  refs.taskName.textContent = effectiveSummary;
  refs.statusLine.textContent = personaSpeechFromSummary(effectiveSummary);
  refs.phaseStrip.querySelectorAll('.phase-pill').forEach((node) => {
    node.classList.toggle('active', node.dataset.phase === current.id);
  });
  controller.renderBeat({
    ...current,
    summary: personaSpeechFromSummary(effectiveSummary),
  });
}

function startAutoPlay() {
  if (state.autoPlayTimer) {
    clearInterval(state.autoPlayTimer);
  }
  if (!state.liveBeats.length) {
    refs.statusLine.textContent = 'Load a run first before auto play.';
    return;
  }
  let index = 0;
  renderLiveBeatAt(index);
  state.autoPlayTimer = setInterval(() => {
    index += 1;
    if (index >= state.liveBeats.length) {
      clearInterval(state.autoPlayTimer);
      state.autoPlayTimer = null;
      return;
    }
    renderLiveBeatAt(index);
  }, 1400);
}

function startRefreshLoop() {
  if (state.refreshTimer) {
    clearInterval(state.refreshTimer);
  }
  state.refreshTimer = setInterval(async () => {
    try {
      const payload = await fetchJSON('/api/frontstage/latest');
      const update = detectFrontstageUpdate(
        {runID: state.liveRunID, beatCount: state.lastSeenBeatCount},
        payload,
      );
      if (!update.hasUpdate) {
        return;
      }

      state.liveRunID = payload.run?.id || null;
      state.liveBeats = payload.timeline || payload.beats || [];
      state.lastSeenBeatCount = state.liveBeats.length;
      refs.runName.textContent = payload.run?.goal || payload.run?.id || 'No run';

      renderInteractionList(
        {summary: payload.presentation?.display_status || 'Updated frontstage payload.'},
        state.liveBeats,
      );

      if (update.reason === 'new_run') {
        refs.statusLine.textContent = 'Detected a new run. Replaying frontstage.';
        startAutoPlay();
        return;
      }

      if (update.reason === 'new_beats' && !state.autoPlayTimer) {
        refs.statusLine.textContent = `Detected ${update.newBeatCount} new beat(s). Playing updates.`;
        startAutoPlayFromIndex(Math.max(0, state.liveBeats.length - update.newBeatCount));
      }
    } catch (error) {
      refs.statusLine.textContent = String(error);
    }
  }, 2500);
}

function renderLiveBeatAt(index) {
  const beat = state.liveBeats[index];
  if (!beat) {
    return;
  }
  state.currentBeat = beat.type;
  renderBeatByType(beat.type, beat.summary);
  renderInteractionList({
    summary: beat.summary,
  }, state.liveBeats.slice(0, index + 1));
}

function startAutoPlayFromIndex(startIndex) {
  if (state.autoPlayTimer) {
    clearInterval(state.autoPlayTimer);
  }
  if (!state.liveBeats.length) {
    return;
  }
  let index = startIndex;
  renderLiveBeatAt(index);
  state.autoPlayTimer = setInterval(() => {
    index += 1;
    if (index >= state.liveBeats.length) {
      clearInterval(state.autoPlayTimer);
      state.autoPlayTimer = null;
      return;
    }
    renderLiveBeatAt(index);
  }, 1400);
}

function renderInteractionList(current = {summary: 'Awaiting start'}, feedOverride = null) {
  refs.interactionList.innerHTML = '';
  const sourceItems = feedOverride
    ? feedOverride.map((item) => ({role: item.actor_role || item.type || 'beat', body: item.summary || item.title || ''}))
    : [
      ...interactionSeed,
      {role: 'beat', body: current.summary},
    ];
  const feed = buildInteractionFeed(sourceItems, 4);
  feed.forEach((item) => {
    const wrapper = document.createElement('div');
    wrapper.className = 'interaction-item';
    wrapper.innerHTML = `<div class="meta">${item.role}</div><div>${item.body}</div>`;
    refs.interactionList.appendChild(wrapper);
  });
  controller.setInteractionFeed(feed);
}

async function fetchJSON(path) {
  const response = await fetch(path);
  if (!response.ok) {
    throw new Error(`${path}: ${response.status}`);
  }
  return response.json();
}

loadLatestRun().catch((error) => {
  refs.statusLine.textContent = String(error);
});
startRefreshLoop();
