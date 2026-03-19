import test from 'node:test';
import assert from 'node:assert/strict';
import {computeLiveRefreshPlan} from './live.js';

test('computeLiveRefreshPlan refreshes matching session and task', () => {
  const plan = computeLiveRefreshPlan({
    currentTaskID: 'task-2',
    currentSessionID: 'session-1',
  }, {
    id: 'run-1',
    session_id: 'session-1',
    tasks: [
      {id: 'task-1', status: 'done'},
      {id: 'task-2', status: 'in_progress'},
    ],
  });

  assert.equal(plan.nextTaskID, 'task-2');
  assert.equal(plan.refreshSession, true);
  assert.equal(plan.refreshTaskDetail, true);
});

test('computeLiveRefreshPlan falls back to first task when selected task disappears', () => {
  const plan = computeLiveRefreshPlan({
    currentTaskID: 'task-missing',
    currentSessionID: 'session-1',
  }, {
    id: 'run-1',
    session_id: 'session-2',
    tasks: [
      {id: 'task-a', status: 'pending'},
    ],
  });

  assert.equal(plan.nextTaskID, 'task-a');
  assert.equal(plan.refreshSession, false);
  assert.equal(plan.refreshTaskDetail, true);
});

test('computeLiveRefreshPlan handles empty task lists', () => {
  const plan = computeLiveRefreshPlan({
    currentTaskID: 'task-1',
    currentSessionID: 'session-1',
  }, {
    id: 'run-1',
    session_id: 'session-1',
    tasks: [],
  });

  assert.equal(plan.nextTaskID, null);
  assert.equal(plan.refreshTaskDetail, false);
  assert.equal(plan.refreshSession, true);
});
