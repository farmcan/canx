async function loadRuns() {
  const response = await fetch('/api/runs');
  const runs = await response.json();
  const list = document.getElementById('runs');
  list.innerHTML = '';
  runs.forEach((run) => {
    const item = document.createElement('li');
    const button = document.createElement('button');
    button.textContent = `${run.id} · ${run.status} · ${run.goal}`;
    button.onclick = () => loadRun(run.id);
    item.appendChild(button);
    list.appendChild(item);
  });
}

async function loadRun(runID) {
  const [runResponse, eventsResponse] = await Promise.all([
    fetch(`/api/runs/${runID}`),
    fetch(`/api/runs/${runID}/events`)
  ]);
  const run = await runResponse.json();
  const events = await eventsResponse.json();
  document.getElementById('run-detail').textContent = JSON.stringify(run, null, 2);
  document.getElementById('events').textContent = JSON.stringify(events, null, 2);
}

loadRuns();
