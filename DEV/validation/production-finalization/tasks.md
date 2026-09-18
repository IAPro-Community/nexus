# Production finalization task graph

| ID | Lane | Scope | Parallel-safe | Verification |
| --- | --- | --- | --- | --- |
| INV | Lead | Baseline, blockers, integration, final report | No | `git status`, gates, same-SHA ledger |
| BE | Backend reviewer | Go runtime, missions, provider, Maestro independence | Yes | focused Go tests, `go vet`, race |
| SEC | Security reviewer | auth, Origin, WebSocket, paths, secrets, updater | Yes | existing security gates and tests |
| WEB | Frontend reviewer | i18n, inline styles, type/lint/style/build/browser | Yes | Web quality and browser scripts |
| WIN | Windows/release reviewer | CI, native jobs, PowerShell, Wails, installer/updater | Yes | workflow/static audit plus native CI evidence |
| FIX | Lead | Integrate only reproduced findings | No | focused regression then full gates |
| REVIEW | Independent reviewers | Reject-oriented final review | Yes | review findings with paths/severity |
| DOC | Lead | Final evidence and release verdict | No | `FINAL_REPORT.md`, docs verification |

Parallel lanes must not edit the same files. Review-only lanes report findings;
the lead owns integration and evidence publication.
