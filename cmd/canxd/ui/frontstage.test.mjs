import test from 'node:test';
import assert from 'node:assert/strict';
import {
  DEFAULT_DEMO_BEATS,
  nextDemoState,
  buildInteractionFeed,
  beatFromPresentation,
  detectFrontstageUpdate,
  personaSpeechFromSummary,
} from './frontstage-core.js';
import {assertFrontstageDriver, createFrontstageController} from './frontstage-driver.js';

test('nextDemoState starts from first phase when idle', () => {
  const state = nextDemoState({
    isPlaying: false,
    currentBeat: null,
    tick: 0,
  });

  assert.equal(state.isPlaying, true);
  assert.equal(state.currentBeat, DEFAULT_DEMO_BEATS[0].id);
  assert.equal(state.currentZone, DEFAULT_DEMO_BEATS[0].zone);
});

test('nextDemoState advances through demo phases', () => {
  const first = nextDemoState({
    isPlaying: true,
    currentBeat: 'briefing',
    tick: 1,
  });
  const second = nextDemoState({
    isPlaying: true,
    currentBeat: first.currentBeat,
    tick: first.tick,
  });

  assert.equal(first.currentBeat, 'tool_use');
  assert.equal(second.currentBeat, 'build');
});

test('buildInteractionFeed keeps latest interaction items first', () => {
  const feed = buildInteractionFeed([
    {role: 'system', body: 'Run queued'},
    {role: 'human', body: 'Please prioritize review'},
    {role: 'agent', body: 'Validation passed'},
    {role: 'agent', body: 'Sync complete'},
  ], 3);

  assert.equal(feed.length, 3);
  assert.equal(feed[0].body, 'Sync complete');
  assert.equal(feed[2].body, 'Please prioritize review');
});

test('beatFromPresentation maps current frontstage phase to generic beat', () => {
  assert.equal(beatFromPresentation({phase: 'working'}), 'tool_use');
  assert.equal(beatFromPresentation({phase: 'blocked'}), 'incident');
  assert.equal(beatFromPresentation({phase: 'done'}), 'complete');
  assert.equal(beatFromPresentation({phase: 'planning'}), 'briefing');
});

test('createFrontstageController renders selected beat through driver interface', () => {
  const calls = [];
  const controller = createFrontstageController(assertFrontstageDriver({
    mount() {
      calls.push('mount');
    },
    render(beat) {
      calls.push(`render:${beat.id}`);
    },
    setInteractionFeed(items) {
      calls.push(`feed:${items.length}`);
    },
  }), {
    beats: DEFAULT_DEMO_BEATS,
  });

  controller.mount();
  const beat = controller.showBeat('review');
  controller.renderBeat({...beat, summary: 'review in progress'});
  controller.setInteractionFeed([{body: 'hi'}]);

  assert.equal(beat.id, 'review');
  assert.deepEqual(calls, ['mount', 'render:review', 'render:review', 'feed:1']);
});

test('detectFrontstageUpdate detects a new run', () => {
  const result = detectFrontstageUpdate(
    {runID: 'run-1', beatCount: 3},
    {run: {id: 'run-2'}, beats: [{id: 'b1'}]},
  );

  assert.equal(result.hasUpdate, true);
  assert.equal(result.reason, 'new_run');
});

test('detectFrontstageUpdate detects appended beats on same run', () => {
  const result = detectFrontstageUpdate(
    {runID: 'run-1', beatCount: 2},
    {run: {id: 'run-1'}, beats: [{id: 'b1'}, {id: 'b2'}, {id: 'b3'}]},
  );

  assert.equal(result.hasUpdate, true);
  assert.equal(result.reason, 'new_beats');
  assert.equal(result.newBeatCount, 1);
});

test('personaSpeechFromSummary converts raw summaries into agent speech', () => {
  assert.equal(
    personaSpeechFromSummary('Validation passed with 12 checks green.'),
    '我这轮的结果是：Validation passed with 12 checks green.',
  );
  assert.equal(
    personaSpeechFromSummary(''),
    '我正在推进当前步骤。',
  );
});
