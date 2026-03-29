import {
  SAMPLE_BEATS,
  nextSamplePlayback,
  beatProgressLabel,
  buildSampleTimeline,
  sampleSpeechFromBeat,
  buildSampleCast,
  buildStageAgents,
} from './frontstage-sample-core.js';

const state = {
  index: -1,
  isPlaying: false,
  timer: null,
};

const refs = {
  actor: document.getElementById('sample-actor'),
  actorRole: document.getElementById('actor-role'),
  beatLabel: document.getElementById('sample-beat-label'),
  beatSummary: document.getElementById('sample-beat-summary'),
  beatSpeech: document.getElementById('sample-beat-speech'),
  beatProgress: document.getElementById('sample-beat-progress'),
  statusLine: document.getElementById('sample-status-line'),
  timeline: document.getElementById('sample-timeline'),
  stage: document.getElementById('sample-stage'),
  cast: document.getElementById('sample-cast'),
  stageAgents: document.getElementById('sample-stage-agents'),
};

const zoneAnchors = {
  command: {x: '16%', y: '22%'},
  forge: {x: '56%', y: '26%'},
  lab: {x: '22%', y: '63%'},
  incident: {x: '50%', y: '68%'},
  sync: {x: '78%', y: '62%'},
};

function renderTimeline() {
  refs.timeline.innerHTML = '';
  buildSampleTimeline().forEach((beat, index) => {
    const item = document.createElement('button');
    item.type = 'button';
    item.className = 'sample-timeline-item';
    item.dataset.beat = beat.id;
    item.innerHTML = `
      <span class="sample-timeline-progress">${beat.progress}</span>
      <strong>${beat.label}</strong>
      <span>${beat.summary}</span>
    `;
    item.onclick = () => {
      stopAutoPlay();
      state.index = index - 1;
      advance();
    };
    refs.timeline.appendChild(item);
  });
}

function setBeat(beat, index) {
  const cast = buildSampleCast(beat);
  refs.actor.dataset.zone = beat.zone;
  refs.actor.dataset.mood = beat.mood;
  refs.stage.dataset.zone = beat.zone;
  refs.actor.style.setProperty('--actor-x', zoneAnchors[beat.zone].x);
  refs.actor.style.setProperty('--actor-y', zoneAnchors[beat.zone].y);
  refs.stage.style.setProperty('--actor-x', zoneAnchors[beat.zone].x);
  refs.stage.style.setProperty('--actor-y', zoneAnchors[beat.zone].y);
  refs.actorRole.textContent = beat.actorTitle;
  refs.beatLabel.textContent = beat.label;
  refs.beatSummary.textContent = beat.summary;
  refs.beatSpeech.textContent = sampleSpeechFromBeat(beat);
  refs.beatProgress.textContent = beatProgressLabel(index);
  refs.statusLine.textContent = `${beat.actorTitle}：${sampleSpeechFromBeat(beat)}`;
  renderCast(cast);
  renderStageAgents(buildStageAgents(beat));

  refs.timeline.querySelectorAll('.sample-timeline-item').forEach((node) => {
    node.classList.toggle('active', node.dataset.beat === beat.id);
  });
}

function renderStageAgents(agents) {
  refs.stageAgents.innerHTML = '';
  agents.forEach((agent) => {
    const node = document.createElement('div');
    node.className = 'sample-stage-agent';
    if (agent.active) {
      node.classList.add('active');
    }
    node.style.left = agent.anchor.x;
    node.style.top = agent.anchor.y;
    node.innerHTML = `
      <div class="sample-stage-agent-sprite" data-role="${agent.id}">
        <div class="sample-stage-agent-head"></div>
        <div class="sample-stage-agent-body"></div>
      </div>
      <div class="sample-stage-agent-label">${agent.title}</div>
    `;
    refs.stageAgents.appendChild(node);
  });
}

function renderCast(cast) {
  refs.cast.innerHTML = '';
  cast.forEach((agent) => {
    const card = document.createElement('div');
    card.className = 'sample-cast-card';
    if (agent.active) {
      card.classList.add('active');
    }
    card.innerHTML = `
      <div class="sample-cast-sprite">${agent.title.slice(0, 1)}</div>
      <div>
        <strong>${agent.title}</strong>
        <div class="sample-cast-meta">${agent.zone}${agent.active ? ' · active' : ''}</div>
      </div>
    `;
    refs.cast.appendChild(card);
  });
}

function advance() {
  const next = nextSamplePlayback(state, SAMPLE_BEATS);
  state.index = next.index;
  state.isPlaying = true;
  setBeat(next.beat, next.index);
}

function stopAutoPlay() {
  if (state.timer) {
    clearInterval(state.timer);
    state.timer = null;
  }
}

function startAutoPlay() {
  stopAutoPlay();
  advance();
  state.timer = setInterval(() => {
    advance();
  }, 2200);
}

document.getElementById('sample-start').onclick = () => {
  startAutoPlay();
};

document.getElementById('sample-next').onclick = () => {
  stopAutoPlay();
  advance();
};

document.getElementById('sample-replay').onclick = () => {
  stopAutoPlay();
  state.index = -1;
  startAutoPlay();
};

renderTimeline();
startAutoPlay();
