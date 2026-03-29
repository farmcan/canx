export function assertFrontstageDriver(driver) {
  const required = ['mount', 'render', 'setInteractionFeed'];
  for (const method of required) {
    if (typeof driver?.[method] !== 'function') {
      throw new Error(`frontstage driver missing method: ${method}`);
    }
  }
  return driver;
}

export function createFrontstageController(driver, options = {}) {
  const beats = options.beats || [];
  let currentBeat = null;

  function findBeat(beatID) {
    return beats.find((beat) => beat.id === beatID) || null;
  }

  return {
    mount() {
      return driver.mount();
    },
    showBeat(beatID) {
      currentBeat = findBeat(beatID);
      if (currentBeat) {
        driver.render(currentBeat);
      }
      return currentBeat;
    },
    renderBeat(beat) {
      currentBeat = beat;
      driver.render(beat);
      return currentBeat;
    },
    setInteractionFeed(items) {
      driver.setInteractionFeed(items);
    },
    getCurrentBeat() {
      return currentBeat;
    },
  };
}
