export function createPhaserFrontstageDriver({Phaser, containerId, zones}) {
  let sceneRef = null;
  let interactionTarget = null;

  const config = {
    type: Phaser.AUTO,
    width: 960,
    height: 640,
    parent: containerId,
    backgroundColor: '#07101d',
    scene: {
      create,
    },
  };

  const game = new Phaser.Game(config);

  function create() {
    sceneRef = this;
    drawScene(this);
  }

  function drawScene(scene) {
    scene.add.rectangle(480, 320, 960, 640, 0x09111f).setStrokeStyle(2, 0x223047);
    scene.add.ellipse(220, 120, 280, 180, 0x11253f, 0.25);
    scene.add.ellipse(720, 160, 320, 220, 0x143024, 0.18);
    scene.add.ellipse(640, 500, 280, 180, 0x2d1f0d, 0.14);

    Object.entries(zones).forEach(([id, zone]) => {
      const zoneRect = scene.add.rectangle(zone.x, zone.y, zone.width, zone.height, 0x0f172a, 0.82)
        .setStrokeStyle(2, 0x334155)
        .setOrigin(0.5);
      const label = scene.add.text(zone.x - zone.width / 2 + 14, zone.y - zone.height / 2 + 14, zone.label, {
        fontFamily: 'Inter, sans-serif',
        fontSize: '14px',
        color: '#cbd5e1',
      });
      zoneRect.name = `zone:${id}`;
      label.name = `label:${id}`;
    });

    const avatar = scene.add.container(180, 150);
    const shadow = scene.add.ellipse(0, 48, 78, 24, 0x000000, 0.32);
    const legLeft = scene.add.rectangle(-14, 24, 12, 34, 0x10243d, 1);
    const legRight = scene.add.rectangle(14, 24, 12, 34, 0x10243d, 1);
    const torso = scene.add.rectangle(0, -2, 48, 52, 0x1d4ed8, 1).setStrokeStyle(2, 0x93c5fd);
    const core = scene.add.rectangle(0, -2, 12, 12, 0x67e8f9, 1);
    const armLeft = scene.add.rectangle(-30, -2, 12, 34, 0x15345a, 1);
    const armRight = scene.add.rectangle(30, -2, 12, 34, 0x15345a, 1);
    const tool = scene.add.rectangle(42, 2, 18, 10, 0xfbbf24, 1);
    const head = scene.add.rectangle(0, -40, 34, 30, 0xbcd8ff, 1).setStrokeStyle(2, 0x60a5fa);
    const visor = scene.add.rectangle(0, -40, 20, 10, 0x10243d, 1);
    const bubble = scene.add.container(110, -20);
    const bubbleBg = scene.add.rectangle(0, 0, 290, 78, 0x0f172a, 0.98).setStrokeStyle(2, 0x60a5fa);
    const bubbleText = scene.add.text(-126, -24, 'Waiting for start...', {
      fontFamily: 'monospace',
      fontSize: '15px',
      wordWrap: {width: 220},
      color: '#e2e8f0',
    });
    bubble.add([bubbleBg, bubbleText]);
    avatar.add([shadow, legLeft, legRight, armLeft, armRight, torso, core, tool, head, visor, bubble]);
    avatar.setName('avatar');
    torso.setName('avatarBody');
    shadow.setName('avatarShadow');
    armLeft.setName('avatarArmLeft');
    armRight.setName('avatarArmRight');
    tool.setName('avatarTool');
    core.setName('avatarCore');
    bubbleText.setName('bubbleText');
  }

  function render(beat) {
    if (!sceneRef) {
      return;
    }
    const avatar = sceneRef.children.getByName('avatar');
    const avatarBody = avatar.getByName('avatarBody');
    const avatarArmLeft = avatar.getByName('avatarArmLeft');
    const avatarArmRight = avatar.getByName('avatarArmRight');
    const avatarTool = avatar.getByName('avatarTool');
    const avatarCore = avatar.getByName('avatarCore');
    const bubbleText = avatar.getByName('bubbleText');
    bubbleText.setText(beat.summary);
    avatarBody.setFillStyle(beat.color, 1);
    avatarCore.setFillStyle(beat.color, 1);
    avatarTool.setFillStyle(beat.id === 'incident' ? 0xef4444 : 0xfbbf24, 1);

    sceneRef.children.list.forEach((child) => {
      if (child.name?.startsWith('zone:')) {
        child.setStrokeStyle(2, 0x334155);
        child.alpha = 1;
      }
    });

    const activeZone = sceneRef.children.getByName(`zone:${beat.zone}`);
    if (activeZone) {
      activeZone.setStrokeStyle(3, beat.color);
    }

    const target = zones[beat.zone] || zones.command_deck;
    sceneRef.tweens.killTweensOf(avatar);
    sceneRef.tweens.killTweensOf(avatarArmLeft);
    sceneRef.tweens.killTweensOf(avatarArmRight);
    sceneRef.tweens.killTweensOf(avatarTool);
    sceneRef.tweens.add({
      targets: avatar,
      x: target.x,
      y: target.y,
      duration: 650,
      ease: 'Sine.InOut',
      yoyo: beat.id === 'tool_use' || beat.id === 'build' || beat.id === 'inspect',
      repeat: beat.id === 'tool_use' || beat.id === 'build' || beat.id === 'inspect' ? -1 : 0,
    });
    sceneRef.tweens.add({
      targets: [avatarArmLeft, avatarArmRight],
      angle: beat.id === 'tool_use' || beat.id === 'build' ? {from: -12, to: 20} : {from: -4, to: 4},
      duration: beat.id === 'tool_use' || beat.id === 'build' ? 320 : 600,
      yoyo: true,
      repeat: -1,
    });
    sceneRef.tweens.add({
      targets: avatarTool,
      alpha: beat.id === 'incident' ? {from: 1, to: 0.38} : {from: 0.85, to: 1},
      duration: beat.id === 'incident' ? 240 : 520,
      yoyo: true,
      repeat: -1,
    });

    if (interactionTarget && beat.id === 'review') {
      sceneRef.tweens.add({
        targets: interactionTarget,
        alpha: 0.55,
        duration: 280,
        yoyo: true,
        repeat: 2,
      });
    }
  }

  function setInteractionFeed(items) {
    if (!sceneRef || items.length === 0) {
      return;
    }
    interactionTarget = sceneRef.children.getByName('zone:review_gate');
  }

  return {
    mount() {
      return game;
    },
    render,
    setInteractionFeed,
  };
}
