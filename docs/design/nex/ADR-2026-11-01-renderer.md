# ADR NEX-001: renderer for the interactive motion candidate

- Date: 2026-11-01
- Status: Accepted for the current prototype; final art acceptance is separate.
- Scope: NEX HUD motion and the private Nexus-Site lab. This decision does not approve public release.

## Context

The current HUD character is an accessible React button containing an SVG assembled from the existing NEX shapes. The Hero source is a layered raster candidate. The chosen pixel-art master, compact/standard exports, and authored 16-view turnaround do not exist yet. A renderer must therefore preserve the current character and Wails hit area, work without WebGL, keep business state outside animation code, and avoid adding a large dependency before real assets exist.

## Options

| Option | Input latency and bundle | Wails/WebView, alpha, 64px precision | Accessibility and maintenance | Decision |
| --- | --- | --- | --- | --- |
| Canvas 2D + sprite atlas | Small runtime; draw work is event/frame driven. Input latency depends on scheduling and hit testing outside the canvas. | Broad WebView support and alpha; excellent nearest-neighbor control once real pixel-art frames exist. | Canvas needs a separate semantic control and renderer lifecycle; good fit for a future atlas, premature for the present layered SVG. | Keep as an interchangeable future implementation behind `NexRenderer`. |
| DOM/SVG + CSS/WAAPI | No rendering dependency; current layers already exist in the DOM. Event feedback uses browser input handlers and bounded CSS transitions. | Works without WebGL and preserves vector edge/alpha behavior. Current SVG is readable at 64px, but it is not the final hand-drawn pixel master. | Native button/focus/ARIA stay with React; selectors target named character parts. Lowest integration and cleanup cost today. | **Selected for current HUD and private lab prototype.** |
| PixiJS v8 | Adds runtime and texture setup; can help batching many atlas sprites, but there is no measured workload demonstrating that benefit here. | Uses WebGL/WebGPU paths where available with fallback considerations; pixel sampling can be controlled. WebView/GPU recovery and texture disposal add lifecycle work. | Canvas still needs a semantic button; another runtime and asset pipeline to maintain. | Reconsider only after an atlas benchmark shows a material benefit. No dependency added. |
| Three.js | Largest runtime and GPU surface among these choices; no valid latency or bundle result exists for this product. | Requires a real model and materials to render real views. Rotating a frontal plane is not a valid turnaround. GPU-disabled and context-loss paths add risk. | Separate accessible controls plus a 3D asset pipeline; no editable, identity-validated model is present. | Rejected for this delivery. Reopen only with an authored `.blend`/`.glb`, angle validation, and measured benefit. |

## Decision

Use React + DOM/SVG layers with CSS keyframes and short transitions for the current prototype. Keep the `NexIdleDirector` as a local scheduler and leave operational mood projection in the existing HUD models. No PixiJS, Three.js, Rive, WebGL requirement, provider polling, or second business-state model is introduced. The site lab consumes the same versioned source assets and remains `noindex`; the public homepage gate is unchanged.

This is an implementation choice, not a claim that the final renderer or pixel-art master is approved. The final atlas may justify Canvas 2D; PixiJS is conditional on an actual comparative benchmark. The 16-view turntable remains authored 2D views, not a rotated plane.

## Evidence and limits

- `npm run art:nex:hybrid:qa` exercises the five raster states and checks bounded action behavior in headless Chromium. It is browser-mechanics evidence, not a cross-browser or artistic approval.
- `npm run art:nex:motion:qa` exercises all 12 candidate actions over real elapsed browser frames, preemption, reduced motion, and 390/768/1280 CSS-pixel viewports. It captures screenshots under `web/dist/nex-motion-preview-qa/`.
- No CPU/GPU, input-latency, long-task, bundle-delta, site LCP, or native Wails measurements have been produced. This ADR makes no performance claims for those metrics.
- Windows native Wails, macOS, Linux/X11, Linux/Wayland, Chrome/Firefox/Safari, and owner visual approval remain pending.

## Consequences

- Existing keyboard, focus, button actions, and native HUD geometry remain owned by React and the current HUD.
- The SVG composition remains a compatibility renderer. It must not be presented as the selected pixel-art master.
- Idle action CSS is a testable motion prototype. The assembled head tilt and highlight-only fin movement remain known fidelity limits.
- Public HUD/site promotion remains blocked until the release-readiness gates contain exact-SHA evidence and human approval.
