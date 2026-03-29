export const SAMPLE_BEATS = [
  {
    id: 'briefing',
    label: 'Mission Briefing',
    zone: 'command',
    mood: 'focus',
    actorTitle: 'AI 指挥员',
    summary: '接收目标，拆分当前任务，准备把工作包送入工坊。',
    speech: '目标收到，开始拆解。',
  },
  {
    id: 'tool_use',
    label: 'Tool Tuning',
    zone: 'forge',
    mood: 'working',
    actorTitle: 'AI 工匠',
    summary: '进入工坊，调优工具，开始一轮真实的 Tool Use。',
    speech: '工具挂载完成，开始搬砖。',
  },
  {
    id: 'inspect',
    label: 'Validation Sweep',
    zone: 'lab',
    mood: 'inspect',
    actorTitle: 'AI 检查员',
    summary: '切到实验台，读取输出，确认这轮结果是否可继续。',
    speech: '指标稳定，继续推进。',
  },
  {
    id: 'incident',
    label: 'Incident Response',
    zone: 'incident',
    mood: 'alert',
    actorTitle: 'AI 值班员',
    summary: '检测到异常，进入告警区，准备介入与修复。',
    speech: '发现异常，先止损再定位。',
  },
  {
    id: 'complete',
    label: 'Sync Complete',
    zone: 'sync',
    mood: 'success',
    actorTitle: 'AI 归档员',
    summary: '本轮交付完成，结果同步，系统回到绿色状态。',
    speech: '本轮完成，已归档。',
  },
];

const SAMPLE_CAST = [
  {id: 'supervisor', title: 'Supervisor', zone: 'command'},
  {id: 'worker', title: 'Worker', zone: 'forge'},
  {id: 'reviewer', title: 'Reviewer', zone: 'lab'},
  {id: 'operator', title: 'Ops', zone: 'sync'},
];

const STAGE_ANCHORS = {
  supervisor: {x: '18%', y: '32%'},
  worker: {x: '60%', y: '36%'},
  reviewer: {x: '28%', y: '74%'},
  operator: {x: '82%', y: '72%'},
};

export function nextSamplePlayback(state, beats = SAMPLE_BEATS) {
  const currentIndex = Number.isInteger(state?.index) ? state.index : -1;
  const nextIndex = (currentIndex + 1) % beats.length;
  const beat = beats[nextIndex];
  return {
    index: nextIndex,
    isPlaying: true,
    beat,
  };
}

export function beatProgressLabel(index, beats = SAMPLE_BEATS) {
  if (index < 0 || index >= beats.length) {
    return '0 / 0';
  }
  return `${index + 1} / ${beats.length}`;
}

export function buildSampleTimeline(beats = SAMPLE_BEATS) {
  return beats.map((beat, index) => ({
    ...beat,
    progress: beatProgressLabel(index, beats),
  }));
}

export function sampleSpeechFromBeat(beat) {
  if (beat?.speech) {
    return beat.speech;
  }
  const normalized = String(beat?.summary || '').replace(/\s+/g, ' ').trim();
  if (!normalized) {
    return '我正在准备下一步。';
  }
  return `我现在在处理：${normalized}`;
}

export function buildSampleCast(activeBeat) {
  const activeAgentID = activeBeat?.zone === 'command'
    ? 'supervisor'
    : activeBeat?.zone === 'forge'
      ? 'worker'
      : activeBeat?.zone === 'lab'
        ? 'reviewer'
        : activeBeat?.zone === 'incident'
          ? 'operator'
          : 'operator';

  return SAMPLE_CAST.map((agent) => ({
    ...agent,
    active: agent.id === activeAgentID,
  }));
}

export function buildStageAgents(activeBeat) {
  return buildSampleCast(activeBeat).map((agent) => ({
    ...agent,
    anchor: STAGE_ANCHORS[agent.id],
  }));
}
