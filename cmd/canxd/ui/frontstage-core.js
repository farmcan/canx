export const DEFAULT_DEMO_BEATS = [
  {
    id: 'briefing',
    zone: 'command_deck',
    label: 'Briefing',
    summary: 'Mission intake: the command deck is framing the next objective.',
    color: 0x60a5fa,
  },
  {
    id: 'tool_use',
    zone: 'workbench',
    label: 'Tool Use',
    summary: 'Tool call engaged: the workbench is tuning the active instrument.',
    color: 0x22c55e,
  },
  {
    id: 'build',
    zone: 'workbench',
    label: 'Build',
    summary: 'Assembly beat: the current output is being shaped into a deliverable.',
    color: 0x10b981,
  },
  {
    id: 'inspect',
    zone: 'test_lab',
    label: 'Inspect',
    summary: 'Inspection beat: the validation lab is measuring the current result.',
    color: 0xeab308,
  },
  {
    id: 'review',
    zone: 'review_gate',
    label: 'Review',
    summary: 'Review beat: the gate is deciding whether to pass or return the packet.',
    color: 0xf97316,
  },
  {
    id: 'handoff',
    zone: 'sync_port',
    label: 'Handoff',
    summary: 'Handoff beat: the sync port is packaging results for the next participant.',
    color: 0x14b8a6,
  },
  {
    id: 'incident',
    zone: 'incident_zone',
    label: 'Incident',
    summary: 'Incident beat: the system raised a warning and is waiting for intervention.',
    color: 0xef4444,
  },
  {
    id: 'complete',
    zone: 'sync_port',
    label: 'Complete',
    summary: 'Completion beat: the run is archived and systems are green.',
    color: 0x22c55e,
  },
];

export function nextDemoState(state, beats = DEFAULT_DEMO_BEATS) {
  const currentIndex = beats.findIndex((beat) => beat.id === state.currentBeat);
  const nextIndex = currentIndex === -1 ? 0 : (currentIndex + 1) % beats.length;
  const nextBeat = beats[nextIndex];
  return {
    isPlaying: true,
    currentBeat: nextBeat.id,
    currentZone: nextBeat.zone,
    tick: (state.tick || 0) + 1,
  };
}

export function buildInteractionFeed(messages, limit = 4) {
  return [...messages].reverse().slice(0, limit);
}

export function beatFromPresentation(presentation) {
  switch (presentation.phase) {
    case 'working':
      return 'tool_use';
    case 'blocked':
      return 'incident';
    case 'done':
      return 'complete';
    default:
      return 'briefing';
  }
}

export function detectFrontstageUpdate(previous, payload) {
  const nextRunID = payload?.run?.id || null;
  const nextBeatCount = payload?.beats?.length || 0;
  if (!nextRunID) {
    return {
      hasUpdate: false,
      reason: 'empty',
      newBeatCount: 0,
    };
  }
  if (previous.runID !== nextRunID) {
    return {
      hasUpdate: true,
      reason: 'new_run',
      newBeatCount: nextBeatCount,
    };
  }
  if (nextBeatCount > previous.beatCount) {
    return {
      hasUpdate: true,
      reason: 'new_beats',
      newBeatCount: nextBeatCount - previous.beatCount,
    };
  }
  return {
    hasUpdate: false,
    reason: 'no_change',
    newBeatCount: 0,
  };
}

export function personaSpeechFromSummary(summary) {
  const normalized = String(summary || '').replace(/\s+/g, ' ').trim();
  if (!normalized) {
    return '我正在推进当前步骤。';
  }
  const trimmed = normalized.length > 88 ? `${normalized.slice(0, 85)}...` : normalized;
  return `我这轮的结果是：${trimmed}`;
}
