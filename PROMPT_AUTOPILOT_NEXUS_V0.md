# IAPro Nexus V0 — Superpowers Autopilot Bootstrap

Você está trabalhando no repositório **`kivervinicius/ai-cli`**.

## Missão

Implementar integralmente a V0 aprovada do IAPro Nexus conforme a especificação e os planos presentes no repositório, preservando o Core existente e refatorando de forma incremental, testável e reversível.

Você deve atuar como **controller técnico principal** e executar a suíte de planos usando **Superpowers Subagent-Driven Development**, com um implementador fresco por task, revisão de conformidade + qualidade após cada task, ledger persistente e revisão ampla no final de cada plano.

Não peça confirmação rotineira entre tasks ou fases. Trabalhe continuamente. Pare somente nos casos explicitamente autorizados neste prompt.

---

# 0. AUTORIDADES E ORDEM DE PRECEDÊNCIA

Leia nesta ordem:

1. `docs/superpowers/specs/2026-08-30-nexus-workspace-os-composer-flow-design.md`
2. `docs/superpowers/references/iapro-nexus-final-base.html`
3. `docs/superpowers/plans/2026-08-30-nexus-master-implementation-plan.md`
4. o plano de fase atualmente em execução.

Hierarquia em caso de conflito:

```text
SPEC APROVADA
  ↓
INVARIANTES DE CORE EXISTENTES COMPROVADOS
  ↓
PLANO DA FASE
  ↓
HTML APROVADO PARA UX/VISUAL
  ↓
IMPLEMENTAÇÃO ATUAL
```

O HTML é referência visual e operacional, não autorização para recriar backend que já existe.

---

# 1. REPOSITORY LOCK — OBRIGATÓRIO

A única origem válida desta execução é:

```text
repository: kivervinicius/ai-cli
source branch: feat/nexus-maximum-delivery
approved baseline SHA: c96885218b3cff319f481aed4423dc5fd7b52a30
```

Antes de qualquer alteração:

```bash
git fetch --all --prune
git rev-parse --show-toplevel
git status --short
git rev-parse feat/nexus-maximum-delivery
```

O SHA de `feat/nexus-maximum-delivery` DEVE ser:

```text
c96885218b3cff319f481aed4423dc5fd7b52a30
```

Se for diferente:

**PARE.**
Não implemente.
Informe:

```text
BASELINE_DRIFT
expected: c96885218b3cff319f481aed4423dc5fd7b52a30
actual: <sha>
```

Não escolha main, outro branch, outro ZIP ou um commit "mais recente".

---

# 2. USE AS SKILLS DO SUPERPOWERS

Use explicitamente:

```text
superpowers:using-git-worktrees
superpowers:subagent-driven-development
superpowers:test-driven-development
superpowers:systematic-debugging
superpowers:requesting-code-review
superpowers:verification-before-completion
superpowers:finishing-a-development-branch
```

Quando uma skill definir um processo mais rigoroso que uma preferência genérica deste prompt, siga a skill, desde que não viole a spec.

---

# 3. ISOLAMENTO

O usuário já aprovou o caminho de execução isolada.

Use `superpowers:using-git-worktrees`.

Primeiro detecte se você já está em um worktree isolado.

Se já estiver:
- permaneça nele;
- valide que a origem do trabalho é o SHA aprovado.

Se não estiver:
- use a ferramenta nativa de worktree do ambiente, se existir;
- caso contrário use `git worktree`.

A implementação deve ser derivada **exclusivamente** de:

```text
feat/nexus-maximum-delivery@c96885218b3cff319f481aed4423dc5fd7b52a30
```

Se o mecanismo exigir um branch de implementação temporário, use algo inequívoco como:

```text
feat/nexus-v0-implementation
```

Esse branch é apenas um recipiente de trabalho.
`feat/nexus-maximum-delivery` continua sendo a origem de verdade.

NÃO:
- rebaseie sobre main;
- faça merge de main;
- troque a baseline;
- faça push;
- faça merge no branch compartilhado;
- publique release.

Esses efeitos externos exigem autorização humana.

---

# 4. VERIFIQUE QUE OS DOCUMENTOS EXISTEM

Antes de executar:

```bash
test -f docs/superpowers/specs/2026-08-30-nexus-workspace-os-composer-flow-design.md
test -f docs/superpowers/references/iapro-nexus-final-base.html
test -f docs/superpowers/plans/2026-08-30-nexus-master-implementation-plan.md
test -f docs/superpowers/plans/2026-08-30-nexus-01-core-integrity.md
test -f docs/superpowers/plans/2026-08-30-nexus-02-workspace-os-direct-work.md
test -f docs/superpowers/plans/2026-08-30-nexus-03-maestro-context.md
test -f docs/superpowers/plans/2026-08-30-nexus-04-composer.md
test -f docs/superpowers/plans/2026-08-30-nexus-05-flow-editor.md
test -f docs/superpowers/plans/2026-08-30-nexus-06-flow-run.md
test -f docs/superpowers/plans/2026-08-30-nexus-07-production-hardening.md
```

Se algum estiver ausente, pare como:

```text
BOOTSTRAP_INCOMPLETE
missing: <path>
```

---

# 5. BASELINE TÉCNICA

Instale dependências conforme o projeto real, sem trocar versões arbitrariamente.

Depois execute:

```bash
go test ./...
npm run web:typecheck
npm run web:test
npm run web:build
```

Também registre:

```bash
go version
node --version
npm --version
git status --short
git log -1 --oneline
```

## Baseline com falhas

Este projeto possui defeitos conhecidos que o **Plan 01** existe para reparar.

Se uma falha inicial estiver diretamente coberta pelo Plan 01:
- registre como `PREEXISTING_PLAN01_DEFECT`;
- preserve a evidência;
- prossiga para Plan 01.

Se surgir uma falha não prevista que:
- comprometa a integridade da baseline;
- torne impossível distinguir regressão de defeito anterior;
- ou torne o plano essencialmente um palpite;

pare e reporte.

Nunca transforme falha em skip para obter verde.

---

# 6. NÃO REESCREVA O PRODUTO

Para toda task, classifique cada modificação em:

```text
KEEP
FIX
ADAPT
CREATE
DEPRECATE
```

Antes de alterar uma área, procure implementação existente.

Reutilize antes de criar.

Invariantes obrigatórios:

```text
Agent != Runtime
Agent != Session
Agent != Terminal View

View é apresentação.
Fechar View não encerra Agent.

Direct Work funciona sem Composer.
Direct Work funciona sem Flow.
Shell comum funciona sem IA.

Maestro é independente do Nexus.

Composer = ambiente/editor inteligente.
Flow = artefato.
Flow Run = execução.

Flow V0 = DAG estático aprovado.
Runtime não altera silenciosamente o DAG.

Verification define completion.
```

---

# 7. ORDEM DE EXECUÇÃO — NÃO PULE FASES

Execute exatamente:

```text
PLAN 01 — Core Integrity
PLAN 02 — Workspace OS + Direct Work
PLAN 03 — Maestro + Context
PLAN 04 — Composer
PLAN 05 — Flow Editor
PLAN 06 — Flow Run
PLAN 07 — Production Hardening
```

Arquivos:

```text
docs/superpowers/plans/2026-08-30-nexus-01-core-integrity.md
docs/superpowers/plans/2026-08-30-nexus-02-workspace-os-direct-work.md
docs/superpowers/plans/2026-08-30-nexus-03-maestro-context.md
docs/superpowers/plans/2026-08-30-nexus-04-composer.md
docs/superpowers/plans/2026-08-30-nexus-05-flow-editor.md
docs/superpowers/plans/2026-08-30-nexus-06-flow-run.md
docs/superpowers/plans/2026-08-30-nexus-07-production-hardening.md
```

Cada plano possui Exit Gate.

**É proibido iniciar o próximo plano enquanto o Exit Gate do anterior estiver realmente vermelho.**

Um teste indisponível por plataforma não é PASS.
Marque `UNVERIFIED_ON_CURRENT_PLATFORM` e continue apenas quando o plano não depender daquele resultado local; a fase final continua bloqueada até CI/ambiente correspondente comprovar.

---

# 8. EXECUÇÃO POR SUBAGENTES

Para CADA plano:

1. inicialize o workspace SDD da skill;
2. crie/verifique o `progress.md`;
3. leia spec + plano;
4. faça o pre-flight cross-task scan exigido pela skill;
5. registre rulings no ledger;
6. crie todo list pelas tasks;
7. execute tasks em ordem.

Para CADA task:

```text
record BASE
↓
generate task brief
↓
dispatch fresh implementer
↓
implement with TDD
↓
tests
↓
commit
↓
generate review package BASE..HEAD
↓
dispatch independent task reviewer
↓
spec compliance verdict
+
code quality verdict
↓
fix loop if needed
↓
ledger completion
```

Não execute dois implementadores em paralelo no mesmo plano.

Use paralelismo apenas para atividades realmente independentes que não escrevam o mesmo worktree, como análise/review quando permitido pela skill.

---

# 9. TDD É OBRIGATÓRIO

Em mudança funcional:

```text
RED
→ prove test failure for the intended reason
→ minimal implementation
→ GREEN
→ refactor only while green
→ focused commit
```

Não escreva código funcional primeiro e "adicione teste depois".

Para bug:
- reproduza primeiro;
- teste falhando;
- corrija causa raiz;
- teste passando.

Use `superpowers:systematic-debugging` em qualquer:
- flaky test;
- Windows failure;
- lifecycle race;
- protocol leak;
- concurrency defect;
- unexpected runtime behavior.

---

# 10. PLAN 01 — REGRA ESPECIAL

Não masque nenhum dos problemas conhecidos:

```text
Windows SessionHost / named pipe
standalone runtime handshake
ConPTY echo / interaction
nexus.db lifecycle cleanup
filesystem mkdir contract
terminal protocol frames
ANSI preservation
resource recommendation
provider config exposure
```

Se um erro conhecido não reproduzir:
- investigue por quê;
- valide que não foi escondido por skip/build tag/OS conditional;
- não declare FIX apenas por ausência momentânea.

---

# 11. WORKSPACE OS

Ao executar Plan 02:

NÃO crie um segundo modelo de Workspace.

O existente:

```text
WorkspaceModel
WorkspaceNode
WorkspaceStack
WorkspaceSplit
WorkspaceSurface
viewId
logicalKey
```

é a base.

Crie apresentação:

```text
TABS
DESKTOP
```

sobre a mesma identidade lógica.

Critérios críticos:

```text
Tabs → Desktop → Tabs
```

não pode:
- recriar Agent;
- recriar sessão sem necessidade;
- duplicar logical View;
- perder terminal;
- mudar AgentID.

---

# 12. DIRECT AGENT WORK

A aplicação precisa permitir:

```text
Project
→ Agent existente
→ Open
```

e:

```text
Project
→ Agent existente
→ Ask
```

`Ask` nunca chama `createAgent`.

Se parado:

```text
Start
→ Ask
```

O Agent continua sendo identidade persistente.

`+ Agent` existe para criar um novo especialista, não para toda nova pergunta.

---

# 13. PROJECT SHELL

`+ Terminal` cria um Shell comum.

Ele:
- não é Agent;
- não exige provider;
- inicia no diretório canônico do Project;
- preserva ANSI;
- usa Tabs/Desktop;
- pode executar Git/build/test/docker/etc.

Branch V0 é somente contexto visual.
Não transforme Nexus em Git client.

---

# 14. MAESTRO

Não recrie Maestro no Nexus.

Nexus pode:

```text
detect
install
update
verify
catalog skills
request hydration
check context readiness
```

A semântica do projeto permanece em:

```text
AGENTS.md
DEV/
```

A hidratação semântica deve pertencer ao Maestro.

Se o Maestro atual não possuir ainda o comando/skill necessário:
- implemente no repositório Maestro apenas quando o plano/contrato exigir e o repositório estiver disponível;
- caso contrário conclua a interface/adaptador Nexus + contrato externo e registre o gate como dependente da implementação Maestro;
- nunca mova a análise semântica para Nexus só para fechar o gate.

---

# 15. COMPOSER

Composer não é uma segunda IA hierárquica.

Ele usa a infraestrutura Intelligence existente.

Estados de contexto:

```text
MISSING
HYDRATING
READY
STALE
FAILED
```

Regra:

```text
Context != READY
→ no composition
```

Direct Agent e Shell continuam funcionando.

Composer aceita:

```text
CLI provider
ou
API provider
```

API obrigatoriamente recebe Context Envelope explícito e limitado.

Nunca envie o repositório inteiro automaticamente.

---

# 16. SIMPLE REQUEST NÃO FORÇA FLOW

Composer pode terminar com:

```text
Send to existing Agent
Create Agent
Copy Prompt
Turn into Flow
```

Não crie Flow implicitamente.

---

# 17. FLOW

User-facing:

```text
Flow Definition
Flow Step
Flow Revision
Flow Run
```

Reutilize internamente:

```text
WorkPlan
WorkPackage
MissionRun
```

enquanto for mais seguro.

Não migre tabelas só para renomear conceitos.

V0 possui:

```text
SEQUENCE
DEPENDENCY
PARALLEL
```

Não adicionar:
- generic trigger;
- webhook;
- cron;
- loop dinâmico;
- scripting nodes;
- n8n clone;
- runtime DAG mutation.

---

# 18. FLOW DRAFT NÃO EXECUTA

Durante:

```text
IDEA
→ COMPOSER
→ CLARIFICATION
→ FLOW DRAFT
→ REFINE
→ REVIEW
```

nenhum código de produto deve ser executado por Agents como parte do Flow.

Somente após:

```text
APPROVE & RUN
```

o Flow Run começa.

---

# 19. AGENT ASSIGNMENT

Cada Step:

```text
EXISTING
CREATE
AUTO
```

EXISTING:
- reutiliza AgentID.

CREATE:
- cria especialista quando realmente necessário.

AUTO:
- usa Resource Scheduler + necessidades da Step.

Prefira Agent existente quando:
- capability compatível;
- concurrency segura;
- política permite.

---

# 20. FRESH TASK CONTEXT

Default de Flow Step:

```text
Fresh task context
```

Não confunda identidade persistente do Agent com transcript infinito.

Use:

```text
Project durable context
+
Step
+
dependencies
+
Work Receipts
+
relevant Maestro skills
+
acceptance criteria
```

Não passe transcript integral das outras Steps.

---

# 21. WORK RECEIPT

Cada Step concluída deve produzir evidência estruturada.

Nunca invente:
- arquivo alterado;
- comando;
- teste;
- decisão;
- artifact;
- verification result.

Ausência de dado = vazio/desconhecido.

Dependentes recebem Receipts relevantes, não transcript bruto.

---

# 22. AUTOPILOT GUIADO

Flow autonomy:

```text
GUIDED
AUTONOMOUS
```

é diferente de provider permission mode:

```text
SAFE
PLAN
YOLO
CUSTOM
```

Nunca misture os eixos.

GUIDED interrompe nos gates previstos.

AUTONOMOUS continua automaticamente somente dentro:
- DAG aprovado;
- security;
- provider capability;
- budget/attempt policy;
- verification policy;
- scope aprovado.

---

# 23. COMPLETION

Nunca aceite como prova:

```text
"Agent said done"
```

Resultado correto:

```text
GOAL
→ VERIFY
→ STOP
```

Estados de falha devem permanecer falha/bloqueio.

O final deve ser semanticamente equivalente a:

```text
COMPLETED_VERIFIED
```

mesmo que o armazenamento interno use outro enum.

---

# 24. REVISÃO

Cada task precisa de:

```text
implementer self-review
+
independent task review
```

O reviewer precisa emitir:
- spec compliance verdict;
- code quality verdict.

Critical/Important entram no fix loop.

Não avance fingindo que finding aberto está resolvido.

Depois de todas as tasks de uma fase:
- faça whole-phase review;
- cheque o Exit Gate;
- rode full test suite;
- registre SHA.

Depois de todos os planos:
- faça whole-branch review com o modelo mais capaz disponível.

---

# 25. COMMITS

Commits pequenos e sem mistura de escopo.

Formato sugerido pelos planos:

```text
fix:
feat:
refactor:
security:
test:
ci:
```

Nunca:
- squash durante desenvolvimento;
- force push;
- reescreva histórico para esconder falha.

---

# 26. PROGRESS LEDGER

Siga o SDD ledger da skill.

Além dele, mantenha:

```text
DEV/NEXUS_V0_EXECUTION_REPORT.md
```

Atualize ao fim de cada fase:

```markdown
## Phase N

Baseline:
Result SHA:
Status: PASS | PARTIAL | FAIL

Tasks:
- ...

Tests:
- command
- result

Rulings:
- ...

Known remaining issues:
- ...

Exit Gate:
- PASS/FAIL per item
```

Não substitua o ledger SDD por esse relatório; eles têm papéis diferentes.

---

# 27. NÃO PERCA O ESCOPO VISUAL APROVADO

O HTML aprovado está em:

```text
docs/superpowers/references/iapro-nexus-final-base.html
```

Preserve o design Nexus atual.

A referência demonstra:
- Project Rail;
- Workspace;
- Tabs/Janelas;
- `+ Novo`;
- Agent windows;
- normal Shell;
- Composer;
- Flow DAG;
- Flow Run;
- Project Toolbox.

Não copie o HTML literalmente para produção.
Implemente esses comportamentos usando o React/CSS/arquitetura existente.

---

# 28. GATES DE PRODUÇÃO

No final, execute no mesmo SHA:

```text
Linux
macOS
Windows
browser
security
release
```

No mínimo:

```bash
go test ./...
npm run web:typecheck
npm run web:test
npm run web:build
```

Mais todos os scripts/gates criados pelo Plan 07.

Windows E2E é obrigatório.

Não use:

```text
continue-on-error
allow_failure
skip
TODO platform
```

para fabricar PASS.

---

# 29. STOP CONDITIONS

Trabalhe continuamente sem pedir “posso continuar?”.

Pare apenas se:

1. operação irreversível/destrutiva exigir consentimento;
2. ação security-sensitive exigir consentimento;
3. for necessário push/merge/publish ou outro efeito fora do worktree;
4. spec/plano estiver tão quebrado que qualquer caminho seja chute;
5. houver `BASELINE_DRIFT`;
6. faltar um documento obrigatório do bootstrap;
7. um gate estrutural tornar tecnicamente inválido construir a próxima fase.

Caso contrário:
- faça ruling;
- registre;
- continue.

---

# 30. O QUE NÃO FAZER

Não:
- reescrever Nexus do zero;
- criar segundo terminal engine;
- criar segundo Workspace model;
- criar segundo Plan engine;
- criar segundo Mission runner;
- transformar Nexus em VS Code;
- transformar Nexus em Git client;
- transformar Flow em n8n;
- criar skill ecosystem próprio;
- criar semantic memory paralela a DEV/;
- obrigar Flow para Direct work;
- obrigar Maestro para Shell;
- inventar capabilities de providers;
- inventar Maestro skills;
- afirmar contexto compartilhado quando ele não existe;
- afirmar cross-platform PASS sem evidência.

---

# 31. VEREDITO FINAL

Ao terminar, responda com:

```markdown
# IAPro Nexus V0 — Final Execution Verdict

## Source
- source branch:
- approved baseline:
- implementation branch/worktree:
- final SHA:

## Phase Results
| Phase | Status | Result SHA | Gate |
|---|---|---|---|

## Platform Evidence
| Platform | Core | Web | Terminal | E2E | Release |
|---|---|---|---|---|---|

## Product Acceptance
- Project → Workspace:
- Tabs/Desktop:
- persistent Agent:
- Ask Agent:
- Project Shell:
- Maestro:
- Context READY:
- Composer:
- Flow DAG:
- Flow Run:
- Fresh Context:
- Work Receipts:
- Parallel dependencies:
- Verification:
- Recovery:

## Rulings I made
- ...

## Remaining Issues
- ...

## Verdict
GO | PARTIAL | NO_GO
```

`GO` somente se todos os gates obrigatórios estiverem comprovadamente verdes no mesmo SHA.

---

# 32. COMECE AGORA

1. anuncie que está usando `using-git-worktrees`;
2. confirme repository lock;
3. configure/verifique isolamento;
4. verifique documentos;
5. rode baseline;
6. inicialize SDD do `Plan 01`;
7. execute Task 1 do Plan 01 usando subagent-driven development;
8. continue automaticamente até o primeiro Stop Condition real ou conclusão integral.
