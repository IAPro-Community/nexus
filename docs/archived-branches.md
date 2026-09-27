# Archived branch map

Updated: 2026-09-27

The active branch policy for Nexus 1.0 is:

- `main` — repository default branch.
- `release/v1.0.0` — single active Nexus 1.0 release line.
- `archive/2026-09-27/**` — immutable historical aliases created during V1 consolidation.

The GitHub connector available during consolidation cannot delete or rename Git refs, so the legacy names may still appear in the branch list. They are superseded and must not receive new commits. Each legacy ref has an archive alias pointing to the exact same commit SHA.

| Legacy ref | Archived ref | SHA |
| --- | --- | --- |
| `chore/nexus-1.0-post-merge-status` | `archive/2026-09-27/chore/nexus-1.0-post-merge-status` | `491a92b4306c70ee241bfff53a38a03717bd98f5` |
| `codex/windows-stability-pr2-20260917` | `archive/2026-09-27/codex/windows-stability-pr2-20260917` | `17f6f17071d726d2c5000a6ae16bb01e5d8901ea` |
| `codex/windows-terminal-functional` | `archive/2026-09-27/codex/windows-terminal-functional` | `2c44347b10c303397ad15db9b14450d63b0800e5` |
| `feat/agent-interop-ux-projection` | `archive/2026-09-27/feat/agent-interop-ux-projection` | `6ec99f5f89e19bc4392cfb1439f8c988ba159afb` |
| `feat/capacity-monitor-revival` | `archive/2026-09-27/feat/capacity-monitor-revival` | `300ccbeee4f145f80ebce13c9cefafcb01acefc3` |
| `feat/hud-cross-platform-completion` | `archive/2026-09-27/feat/hud-cross-platform-completion` | `68d15c032e2efa87667b064a9df5fd7f5b7a9d64` |
| `feat/native-platform-support` | `archive/2026-09-27/feat/native-platform-support` | `ab05504959ce3a49bae6dc136369c3f259c70677` |
| `feat/nexus-1.0-closure` | `archive/2026-09-27/feat/nexus-1.0-closure` | `0d6e9c667af39b7534002d060848acfc9eb8e265` |
| `feat/nexus-1.1-agent-interop-spec` | `archive/2026-09-27/feat/nexus-1.1-agent-interop-spec` | `23abc1c474c4508f5dc79d5bd9af6243dd3cf55d` |
| `feat/nexus-agent-interop` | `archive/2026-09-27/feat/nexus-agent-interop` | `e44470fb22b1a6050dbc8ae1c0c02d5f5b012ef2` |
| `feat/nexus-agent-interop-ux` | `archive/2026-09-27/feat/nexus-agent-interop-ux` | `b1d96248e8b40c67730dddcd440281a1d615339a` |
| `feat/nexus-hud` | `archive/2026-09-27/feat/nexus-hud` | `609a2b58e3febd9c59a44917e4b9bdf51971955e` |
| `feat/nexus-maximum-delivery` | `archive/2026-09-27/feat/nexus-maximum-delivery` | `294fc6acbeb358fe63d672e673438253a702bba0` |
| `feat/opencode-agent-projection` | `archive/2026-09-27/feat/opencode-agent-projection` | `66b7eed5aa7e44f391839692365db9fb3f0d0909` |
| `feat/v1-beta3-finalization` | `archive/2026-09-27/feat/v1-beta3-finalization` | `78a4493bf34096210d337b4364eebb3dcb4fc338` |
| `feat/v1-opencode-agent-projection` | `archive/2026-09-27/feat/v1-opencode-agent-projection` | `0bef5fe7a824330801a9fbbe498ead13b58db014` |
| `feat/voice-accessibility-1.1` | `archive/2026-09-27/feat/voice-accessibility-1.1` | `38b5ec4f2e88dddba709f641a20ce0d5e8593811` |
| `feat/wails-v3-platform-migration` | `archive/2026-09-27/feat/wails-v3-platform-migration` | `59416d2f9073a5e5a80f2800475bee45835b2982` |
| `fix/beta2-distribution-provenance` | `archive/2026-09-27/fix/beta2-distribution-provenance` | `76694ed47614dee6729bb91c43554b318ba6bffc` |
| `fix/beta2-linux-autopilot-20260925` | `archive/2026-09-27/fix/beta2-linux-autopilot-20260925` | `e4574e4cfd4676032e2284f8a1b2a3c5885e287e` |
| `fix/beta2-mission-final-hardening` | `archive/2026-09-27/fix/beta2-mission-final-hardening` | `a38784489c32f7cc0091ed28eaf58212380ed3d6` |
| `fix/beta2-release-preflight` | `archive/2026-09-27/fix/beta2-release-preflight` | `75aa89584395536c7cdf28fca0f6ccfbc6d08ca2` |
| `fix/beta2-release-truth` | `archive/2026-09-27/fix/beta2-release-truth` | `fc2548b4d13b63f1eb1a571d9525126340fa43f2` |
| `fix/beta2-windows-autopilot-20260925-1039` | `archive/2026-09-27/fix/beta2-windows-autopilot-20260925-1039` | `a9f6dda3ff063b7b46d2260680a75d09cfa661c3` |
| `fix/beta3-homologation-convergence-20260925` | `archive/2026-09-27/fix/beta3-homologation-convergence-20260925` | `ed59fd3e4b079b0b9fa613ebd2bba1be79df9897` |
| `fix/beta3-windows-autopilot-20260926` | `archive/2026-09-27/fix/beta3-windows-autopilot-20260926` | `ce52b09d958862e7f32e33ad1af4821c4a241d7d` |
| `fix/interactive-lead-routing-terminal-keys` | `archive/2026-09-27/fix/interactive-lead-routing-terminal-keys` | `ac0f8fe1ff3f9080dfa87c0b841bb19d9afabb72` |
| `fix/mission-promotion-admission-fallback` | `archive/2026-09-27/fix/mission-promotion-admission-fallback` | `7eaa08d78cebbf907ab653d1e26915c312eefbbe` |
| `fix/nexus-agent-interop-provider-truth` | `archive/2026-09-27/fix/nexus-agent-interop-provider-truth` | `ea43c876d49c4358199becd69980ec5218a776ec` |
| `fix/provider-terminal-transparency` | `archive/2026-09-27/fix/provider-terminal-transparency` | `46bd02b130d36c5a63714d54da55c9b50cd64af9` |
| `fix/v1-beta2-consolidation` | `archive/2026-09-27/fix/v1-beta2-consolidation` | `4d042eb06626cee1323e8d3fb8c3b44b54872d50` |
| `fix/workspace-dir-validation-v1.0.0` | `archive/2026-09-27/fix/workspace-dir-validation-v1.0.0` | `9718e35ce9bcf07726ea2f9fddf75c7ecc03600c` |
| `release/v1.0.0-beta` | `archive/2026-09-27/release/v1.0.0-beta` | `599d0394367a2637823b250df0ac18de891ad537` |
| `release/v1.0.0-beta.2` | `archive/2026-09-27/release/v1.0.0-beta.2` | `f72bdfd4f8476b731b3663f00a672104625833a6` |
| `release/v1.0.0-beta.3` | `archive/2026-09-27/release/v1.0.0-beta.3` | `a0dc145dffb3855fdafd43a91f70fab0717aa08c` |
| `release/v1.0.0-cert` | `archive/2026-09-27/release/v1.0.0-cert` | `f72dae8de92cf27fb3c2540a65dd4db051ddbf4d` |
| `sync/native-platform-support-794933` | `archive/2026-09-27/sync/native-platform-support-794933` | `e8295c7bd06da7e44d776b49a7ddfc5a2204946f` |

Do not branch new beta lines from historical refs. Beta iterations are represented by `VERSION`, tags and signed manifests while development remains on `release/v1.0.0`.
