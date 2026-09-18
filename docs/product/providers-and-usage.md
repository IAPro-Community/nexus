# Providers e usage/quota

Providers são CLIs ou integrações detectadas localmente. O Nexus não instala skills, providers ou credenciais silenciosamente.

| Estado | Significado |
| --- | --- |
| `LIVE` | Dado consultado na fonte agora. |
| `CACHED` | Dado persistido de consulta anterior. |
| `ESTIMATED` | Estimativa explicitamente marcada. |
| `UNKNOWN` | Não há evidência suficiente. |
| `RATE_LIMITED` | A fonte recusou por limite. |
| `UNAVAILABLE` | A fonte não está acessível. |

`UNKNOWN` não é 0%, 100% nem `LIVE`. Consulte `nexus providers` e `nexus usage`.
