# Backward compatibility matrix

| Surface | Before | After this consolidation | Compatibility |
| --- | --- | --- | --- |
| CLI provider aliases | `nexus codex`, `agy`, `claude`, `opencode`, `gemini`, `cursor` | Same dispatch through existing registries | Preserved |
| CLI general commands | Manual dispatch in `internal/app/app.go` | Unchanged in this milestone | Preserved |
| REST paths | `/api/v1/*` registered directly by `Server` | Same paths registered by route groups | Preserved |
| REST response payloads | Existing handler payloads and `error` string | Existing payloads unchanged; additive `/api/v1/system/info` | Preserved/additive |
| Auth/CSRF | Loopback session, bootstrap, cookies, CSRF | Same middleware and handlers | Preserved |
| Web API client | `web/src/nexus/api.ts` plus legacy runtime transport | Nexus facade retained; auth/CSRF/error transport centralized in `web/src/api.ts` | Preserved |
| Provider capabilities | Existing adapter/driver registries | Metadata derived from existing driver registry | Preserved/additive |
| Missions planning API | Existing mission CRUD/task/assignment paths | Same JSON paths, now delegated through `MissionApplicationService` | Preserved |
| MissionRun persistence | Runner repository contract | Same durable records; canceled contexts now fail before I/O/mutation | Preserved/safer |
| Configuration | Existing profile/store files | No migration | Preserved |

No incompatibility was introduced intentionally. Native Windows/macOS runtime
behavior is not claimed from Linux-only execution and remains separately
verified by CI/native runs.
