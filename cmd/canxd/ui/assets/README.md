# Frontstage Assets

This directory is reserved for Frontstage visual assets.

The MVP works without any real image assets. When an asset is missing, the UI
falls back to CSS-only placeholders and zone blocks.

## Recommended files

- `frontstage-scene-bg.webp`
- `zone-command.webp`
- `zone-workbench.webp`
- `zone-test-lab.webp`
- `zone-review-gate.webp`
- `zone-sync-port.webp`
- `zone-incident.webp`

## Avatar state slots

- `avatar-idle.webp`
- `avatar-planning.webp`
- `avatar-working.webp`
- `avatar-validating.webp`
- `avatar-reviewing.webp`
- `avatar-syncing.webp`
- `avatar-blocked.webp`

## Suggested dimensions

- Scene background: `1600x900` or larger, 16:9
- Zone overlays: flexible, keep transparent background
- Avatar states: square, `256x256` or `512x512`, transparent background

## Notes

- Keep all avatar states at the same camera angle and scale.
- Prefer transparent backgrounds for character and effect assets.
- Use WebP or PNG for the first pass.
