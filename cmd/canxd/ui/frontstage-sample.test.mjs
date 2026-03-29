import test from 'node:test';
import assert from 'node:assert/strict';
import {
  SAMPLE_BEATS,
  nextSamplePlayback,
  beatProgressLabel,
  buildSampleTimeline,
  sampleSpeechFromBeat,
  buildSampleCast,
  buildStageAgents,
} from './frontstage-sample-core.js';

test('nextSamplePlayback starts from first beat when idle', () => {
  const next = nextSamplePlayback({index: -1, isPlaying: false});

  assert.equal(next.isPlaying, true);
  assert.equal(next.index, 0);
  assert.equal(next.beat.id, SAMPLE_BEATS[0].id);
});

test('nextSamplePlayback loops through the sample beats', () => {
  const fromLast = nextSamplePlayback({index: SAMPLE_BEATS.length - 1, isPlaying: true});

  assert.equal(fromLast.index, 0);
  assert.equal(fromLast.beat.id, SAMPLE_BEATS[0].id);
});

test('beatProgressLabel returns visible progress', () => {
  assert.equal(beatProgressLabel(0), `1 / ${SAMPLE_BEATS.length}`);
  assert.equal(beatProgressLabel(-1), '0 / 0');
});

test('buildSampleTimeline decorates each beat with progress', () => {
  const timeline = buildSampleTimeline();

  assert.equal(timeline.length, SAMPLE_BEATS.length);
  assert.equal(timeline[1].progress, `2 / ${SAMPLE_BEATS.length}`);
});

test('sampleSpeechFromBeat prefers explicit speech and falls back to summary', () => {
  assert.equal(sampleSpeechFromBeat(SAMPLE_BEATS[0]), '目标收到，开始拆解。');
  assert.equal(
    sampleSpeechFromBeat({summary: '正在比较两次输出差异。'}),
    '我现在在处理：正在比较两次输出差异。',
  );
});

test('buildSampleCast marks the active agent and keeps collaborators visible', () => {
  const cast = buildSampleCast(SAMPLE_BEATS[1]);

  assert.equal(cast.length, 4);
  assert.equal(cast.find((item) => item.id === 'worker').active, true);
  assert.equal(cast.find((item) => item.id === 'supervisor').active, false);
  assert.equal(cast.find((item) => item.id === 'reviewer').zone, 'lab');
});

test('buildStageAgents maps collaborators onto stage anchors', () => {
  const agents = buildStageAgents(SAMPLE_BEATS[0]);

  assert.equal(agents.length, 4);
  assert.equal(agents.find((item) => item.id === 'supervisor').anchor.x, '18%');
  assert.equal(agents.find((item) => item.id === 'worker').anchor.y, '36%');
  assert.equal(agents.find((item) => item.id === 'supervisor').active, true);
});
