<p align="center">
  <img src="nexus-logo.png" alt="IAPro Nexus" width="320">
</p>

<h1 align="center">IAPro Nexus</h1>

<p align="center"><strong>Uma workstation local e um control plane para coding agents.</strong></p>

<p align="center">
Transforme objetivos em trabalho executável, coordene Agents persistentes, escolha recursos de IA,
execute com isolamento e só conclua quando houver evidência verificável.
</p>

<p align="center">
  <a href="docs/quick-start.md">Quick Start</a> ·
  <a href="docs/product/how-it-works.md">Como funciona</a> ·
  <a href="docs/README.md">Documentação</a> ·
  <a href="README.en.md">English</a> ·
  <a href="README.es.md">Español</a>
</p>

![Visão geral do Nexus v1](docs/assets/diagrams/nexus-v1-overview.svg)

> **Estado da v1:** o source está no programa de fechamento do Nexus 1.0 e reporta
> `1.0.0-beta.4`. Isso **não** significa que uma release estável v1 esteja
> certificada ou publicada. A verdade autoritativa está em
> [RELEASE_STATUS](DEV/validation/current/RELEASE_STATUS.md).

## TL;DR

O Nexus reúne em um único Core:

- Projects e workspaces;
- Agents persistentes;
- providers, perfis, modelos e quota;
- terminais e runtimes supervisionados;
- Composer, WorkPlan e Flow;
- Missions duráveis;
- worktrees isolados;
- Scheduler e políticas de recurso;
- memória, receipts, artifacts e evidência;
- recovery, intervenção humana e Definition of Done global.

Para uma tarefa simples, use **Direct**. Para um objetivo complexo, o caminho
canônico é:

```text
Objetivo → Composer → WorkPlan → Flow → GO → MissionRunner → Agents → Evidência → COMPLETED_VERIFIED
```

→ [Entenda o fluxo ponta a ponta](docs/product/how-it-works.md)

## Comece em 5–10 minutos

O caminho recomendado para primeiro uso é **Direct**:

```text
Project → + New → AI Session → Provider → Terminal
```

### Release publicada

A documentação de instalação usa a última beta publicada indicada pelo
[status de release](DEV/validation/current/RELEASE_STATUS.md). Enquanto a v1
estável não estiver certificada, não confunda versão do source com release
publicada.

**Linux / macOS**

```bash
curl -fsSL https://raw.githubusercontent.com/IAPro-Community/nexus/main/install.sh | bash -s -- --version=v1.0.0-beta.1
nexus doctor
nexus web
```

**Windows PowerShell**

```powershell
irm https://raw.githubusercontent.com/IAPro-Community/nexus/main/install.ps1 -OutFile install.ps1
.\install.ps1 -Version v1.0.0-beta.1
nexus doctor
nexus web
```

### Compilar do source

Requisitos de desenvolvimento: Go 1.25+, Bun 1.3.9+, Git e Make.

```bash
git clone https://github.com/IAPro-Community/nexus.git
cd nexus
bun --cwd web install --frozen-lockfile
make build
./nexus doctor
./nexus web
```

Depois:

1. adicione um projeto;
2. use **+ New → AI Session**;
3. selecione um provider disponível;
4. trabalhe no terminal.

Composer, Flow, Mission e Maestro **não são pré-requisitos** para esse caminho.

→ [Quick Start completo](docs/quick-start.md)

## Como escolher o modo de trabalho

| Situação | Use |
| --- | --- |
| correção pequena, debugging, exploração | **Direct** |
| ideia ainda precisa de clareza | **Composer** |
| quer revisar etapas/dependências | **Flow** |
| trabalho longo, multi-Agent ou com gates | **Mission** |
| quer metodologia, skills e gates externos | **Maestro** opcional |

![Fluxo ponta a ponta](docs/assets/diagrams/nexus-v1-flow.svg)

## O modelo da v1

### Agent ≠ provider

Um **Agent** é uma identidade persistente. Codex, Claude, Gemini, AGY, OpenCode
e outros executores são recursos que podem ser selecionados conforme
capacidade, policy e disponibilidade.

```text
Agent
 ├─ AgentSpec
 ├─ AgentRevision
 └─ RuntimeGeneration
      └─ ProviderSession
```

Isso evita acoplar o papel do trabalhador a um provider específico.

### WorkPlan é a fonte de verdade

Composer transforma intenção em um artefato estruturado. O **WorkPlan** é o
plano semântico/versionado. O **Flow** é a projeção visual e editável desse
mesmo plano.

Não existe um segundo scheduler ou um segundo formato de plano escondido na UI.

### GO é um boundary real

**GO** autoriza uma revisão específica do plano. A partir daí, a Mission executa
essa revisão sob políticas, isolamento, gates e evidência.

### Mission conclui por evidência

O runtime não aceita “done” textual como sucesso. Uma Mission pode exigir:

- testes;
- verificações;
- review independente;
- artifacts;
- critérios de aceite;
- ausência de intervenção pendente;
- Definition of Done global.

O estado final é `COMPLETED_VERIFIED`.

## Coordenação entre Agents

Agents podem solicitar contexto, artifact, revisão ou trabalho auxiliar por um
protocolo fechado:

- `NEED_INFO`;
- `ARTIFACT_REQUEST`;
- `REVIEW_REQUEST`;
- `DELEGATE`.

Toda coordenação passa pelo `MissionRunner`. O Nexus tenta memória/receipts
antes de criar outra execução e bloqueia replay quando não consegue provar
segurança.

![Coordenação entre Agents](docs/assets/diagrams/nexus-v1-agent-coordination.svg)

→ [Como funciona a coordenação](docs/product/agent-coordination.md)

## Segurança e guardrails

A v1 opera com **fail closed**:

- push/deploy não são permissões implícitas;
- reviewers/investigators são read-only por padrão;
- escrita paralela usa worktrees isolados;
- dispatch externo desconhecido não é repetido cegamente;
- ausência de evidência não vira sucesso;
- retries e no-progress são limitados;
- mudanças materiais podem exigir intervenção humana.

→ [Guardrails](docs/architecture/guardrails.md) ·
[Segurança](docs/security.md)

## Nexus e Maestro

```text
Nexus   → execução, Agents, runtimes, providers, workspaces, UX e evidência
Maestro → metodologia, skills, risco, gates e políticas de verificação
```

O [Orquestrador Maestro](https://github.com/IAPro-Community/Orquestrador-Maestro)
é opcional. Sem ele, Direct, Agents, Composer, Flow e Mission continuam sendo
capacidades do Nexus.

## Superfícies

| Superfície | Uso principal |
| --- | --- |
| **Web** | operação visual local, Projects, Agents, Composer, Flow e Missions |
| **Desktop** | shell nativo quando a plataforma possui evidência correspondente |
| **CLI** | automação, diagnóstico, scripting, execução headless e terminal |

Suporte nativo depende de evidência por sistema operacional. Veja
[Suporte de plataformas](docs/operations/platform-support.md).

## Documentação

| Quero… | Documento |
| --- | --- |
| usar agora | [Quick Start](docs/quick-start.md) |
| entender o produto inteiro | [Como funciona](docs/product/how-it-works.md) |
| ver interfaces reais | [Tour visual](docs/product/visual-tour.md) |
| entender Composer | [Composer](docs/product/composer.md) |
| entender Flow | [Flow](docs/product/flow.md) |
| entender Missions | [Missions](docs/product/missions.md) |
| entender Agents | [Agents](docs/product/agents.md) |
| entender coordenação dinâmica | [Coordenação entre Agents](docs/product/agent-coordination.md) |
| entender arquitetura | [Arquitetura](ARCHITECTURE.md) |
| operar/diagnosticar | [Operação](docs/operations/README.md) |
| consultar CLI | [CLI](docs/cli/README.md) |
| verificar funcionalidades | [Matriz](docs/reference/feature-matrix.md) |
| contribuir | [CONTRIBUTING](CONTRIBUTING.md) |

A entrada canônica é [docs/README.md](docs/README.md).

## Honestidade operacional

O Nexus diferencia estados como `LIVE`, `CACHED`, `ESTIMATED`,
`UNKNOWN`, `VERIFIED` e `UNVERIFIED`. Documentação, UI e release gates não
devem transformar ausência de evidência em um resultado positivo.

Capturas reais do produto ficam em `docs/assets/screenshots/` com manifesto.
Diagramas conceituais ficam em `docs/assets/diagrams/`.

## Status de release

A documentação de produto descreve **como o source candidato funciona**. A
autorização para uma release estável exige certificação separada do mesmo SHA,
incluindo plataformas nativas e demais gates definidos no programa de fechamento.

→ [Status autoritativo](DEV/validation/current/RELEASE_STATUS.md)

## Licença

Software proprietário distribuído conforme [LICENSE](LICENSE) e os
[Beta Terms](docs/legal/BETA_TERMS.md). Consulte também
[Privacy](docs/legal/PRIVACY.md) e
[Third-Party Notices](docs/legal/THIRD_PARTY_NOTICES.md).
