# Frontstage Asset Prompts

## Gemini-ready short prompts

These are the shortest ready-to-copy prompts for the first usable visual pack.

### 1. Frontstage scene background

```text
Create a polished 2D game illustration background for an AI orchestration frontstage.
Theme: hybrid of command center and workshop.
View: 3/4 top-down.
Include clear areas for command deck, workbench, validation lab, review gate, sync port, and incident zone.
Style: high-quality 2D game illustration, readable composition, clean silhouettes, not pixel art, not photorealistic.
Mood: calm, intelligent, productive, slightly futuristic.
Color palette: deep navy, muted teal, steel blue, warm amber highlights.
No characters.
No text labels baked into the image.
```

### 2. Main character design

```text
Design a main character for an AI orchestration frontstage UI.
Theme: command-center operator mixed with workshop technician.
View: 3/4 top-down game view.
Style: polished 2D game illustration, clean silhouette, readable at small size, not pixel art, not anime exaggeration.
The character should feel competent, calm, and slightly futuristic.
Outfit: practical control-room jacket, utility belt, compact tool accessories, subtle digital or mechanical details.
Color palette: navy, steel blue, muted teal, with small amber accents.
Transparent or plain clean background.
Create a consistent character design suitable for multiple action states.
```

### 3. Planning state

```text
Create a state illustration for the same character.
State: planning.
Action: standing at a command desk, reviewing blueprint-like panels and assigning work.
View: 3/4 top-down.
Style: polished 2D game illustration.
Same character design, same outfit, same scale, same camera angle.
Mood: calm, analytical, in control.
No background scene, transparent or clean plain background.
```

### 4. Working state

```text
Create a state illustration for the same character.
State: working.
Action: actively tuning tools and operating a technical workbench.
The feeling should be elegant hard work: focused, capable, practical, like "搬砖" in a refined software-operations world.
View: 3/4 top-down.
Style: polished 2D game illustration.
Same character design, same outfit, same scale, same camera angle.
No background scene, transparent or clean plain background.
```

### 5. Blocked state

```text
Create a state illustration for the same character.
State: blocked.
Action: responding to a warning console or malfunctioning machine.
The character is urgent but controlled, diagnosing a problem instead of panicking.
View: 3/4 top-down.
Style: polished 2D game illustration.
Same character design, same outfit, same scale, same camera angle.
Include subtle warning-light energy or alert-tool interaction.
No full background scene, transparent or clean plain background.
```

### 6. Consistency add-on

Append these constraints to every state prompt:

```text
Keep the same character identity across all images.
Maintain the same camera angle and visual scale.
Do not change outfit, proportions, or face design between states.
```

## Style baseline

Use the same visual language for all generated assets:

- 3/4 top-down control-room or workshop scene
- readable silhouette
- clean game-like rendering
- slightly stylized AI operator / technician character
- consistent lighting and camera angle
- no text baked into images unless explicitly requested

## 1. Scene background

```text
Create a wide 16:9 background for an AI orchestration control room mixed with a workshop.
View angle: 3/4 top-down, readable like a simulation game scene.
Include six distinct zones: command deck, workbench, test lab, review gate, sync port, incident zone.
Style: polished indie game environment, not cyberpunk excess, not childish.
Color palette: deep navy, steel blue, muted teal, amber status lights.
No characters in the scene.
High readability, clean floor separation, room layout designed for UI overlays.
```

## 2. Main avatar master prompt

```text
Design a single AI operator character for a software orchestration dashboard.
View angle: 3/4 top-down.
Style: stylized technician / digital foreman, readable silhouette, clean game-ready design.
Outfit: practical control-room workwear, subtle futuristic details, utility belt or tool harness.
Keep proportions, face shape, outfit colors, and camera angle consistent across all future states.
Transparent background.
```

## 3. Planning state

```text
Create a state illustration for the same AI operator character.
State: planning.
Action: standing at a command desk, reviewing blueprint-like panels and assigning tasks.
3/4 top-down view, same character design, same scale, transparent background.
Readable game asset, no background scene.
```

## 4. Working state

```text
Create a state illustration for the same AI operator character.
State: working.
Action: actively building or tuning something at a workbench, using compact tools or keyboard-like controls.
Feeling: focused, productive, "搬砖" but elegant and competent.
3/4 top-down view, same character design, same scale, transparent background.
```

## 5. Validating state

```text
Create a state illustration for the same AI operator character.
State: validating.
Action: monitoring a test console or diagnostics rig, checking bars, indicators, or a measurement panel.
3/4 top-down view, same character design, same scale, transparent background.
```

## 6. Reviewing state

```text
Create a state illustration for the same AI operator character.
State: reviewing.
Action: inspecting results at a review gate, comparing panels, making approve/reject decisions.
Mood: careful, analytical, not aggressive.
3/4 top-down view, same character design, same scale, transparent background.
```

## 7. Syncing state

```text
Create a state illustration for the same AI operator character.
State: syncing.
Action: moving data capsules, archive modules, or glowing transfer containers into a sync port.
3/4 top-down view, same character design, same scale, transparent background.
```

## 8. Blocked state

```text
Create a state illustration for the same AI operator character.
State: blocked.
Action: responding to an incident panel, warning light, or malfunctioning machine.
Mood: urgent but controlled.
3/4 top-down view, same character design, same scale, transparent background.
```

## 9. Effect layer prompts

### Warning effect

```text
Create a transparent effect asset for a warning state in a game UI.
Elements: red-orange alert light, triangular pulse, subtle smoke or glitch sparks.
No character, transparent background.
```

### Sync effect

```text
Create a transparent effect asset for a sync or transfer state in a game UI.
Elements: blue-green energy flow, moving particles, structured data-transfer feel.
No character, transparent background.
```

### Review pass effect

```text
Create a transparent UI effect for approval or pass.
Elements: green status flare, soft pulse, subtle check-mark energy motif.
No text, transparent background.
```

### Review reject effect

```text
Create a transparent UI effect for rejection or failed review.
Elements: red warning flare, sharp light edges, controlled alarm feel.
No text, transparent background.
```

## Output guidance

- Prefer transparent background whenever the asset is not the main scene background.
- Keep the same character identity across all state prompts.
- If Gemini supports reference images, reuse the master character output as reference for every state.
- Generate keyframes first; convert to sprite sheets only after style consistency is acceptable.
