# Validação de runtime e autenticação

Base observada: `release/v1.0.0` em `bf6adf0c80ac3531d7291844512a14a017ab0002`, com worktree dirty. O binário instalado é `1.0.0-beta.10`/Go 1.26.10.

## Reinício após instalação

Com autorização explícita do usuário, os quatro hosts Nexus antigos foram encerrados pelo CLI e reiniciados nos mesmos workspaces, com o mesmo perfil autenticado e `--continue`:

| Provedor/perfil | Workspace | Runtime final | Host PID | Estado |
|---|---|---|---:|---|
| AGY / `kivervinicius-gmail` | `/home/desenvolvedor` | `agy-06GPN101WM51W9041KF5RP06GW` | 562924 | RUNNING |
| Codex / `kiver.omegasistemas` | `/home/desenvolvedor` | `codex-06GPN10H3A17JJPJNYGPSBK9B0` | 563300 | RUNNING |
| Codex / `kivergmail` | `/projetos/vpn-dev-workspace` | `codex-06GPN10YT955VBG5D78QGQN6GR` | 563641 | RUNNING |
| Codex / `kivergmail` | `/projetos/repository/sistemas2` | `codex-06GPN11HZJ2KBEWPDVSQRGBC80` | 563947 | RUNNING |

`nexus ps --json` confirma os quatro em RUNNING; todos os hosts carregados têm SHA-256 `9ecd3ff22cd1d6b099d1963424dc7c8c55302295cef54d5a24b7adc715f9d815`. Desktop (`565592`), HUD (`610408`) e servidor Web (`567528`) também carregaram os binários instalados. Os hashes em `/proc/PID/exe` coincidem com os arquivos da instalação. Resumo sanitizado: `logs/runtimes-final-go1.26.10-bf6adf0c.tsv`; hashes e health checks: `logs/processes-final-go1.26.10-bf6adf0c.log`.

O Core Desktop em `127.0.0.1:43253`, o HUD em `127.0.0.1:34885` e o servidor Web em `127.0.0.1:44065` retornaram `/api/v1/health` HTTP 200 e versão `1.0.0-beta.10`.

## Importação de sessão local

Uma fixture isolada usou uma sessão Codex já autenticada do host para criar perfil temporário. A UI completou a cópia; o destino ficou `auth.json` modo `0600`, e metadados/hash da origem permaneceram inalterados. Nenhuma credencial foi impressa nos logs. O diretório de teste foi isolado e removido após a validação.

O resultado visual da importação não apareceu na janela capturada, embora o perfil tenha sido criado e autenticado. É um defeito P2 de feedback; ver `FIXES_APPLIED.md` e [a captura](screenshots/native-main-import-result-bf6adf0c.png).

## Limite dos fluxos de agentes

O reinício comprovou preservação/retomada das sessões de terminal dos quatro hosts. Não foi enviada tarefa sintética às sessões reais em andamento, nem foram consumidos tokens/quota para simular `DIRECT`, `STICKY_LEAD`, `ORCHESTRATED`, `KEEP`, `DELEGATE`, `MISSION`, failover ou invocação de skills. Esses comportamentos seguem `NOT_RUN`; uma invocação externa controlada sem credencial/conta de teste foi classificada `BLOCKED`, não aprovada por inferência.
