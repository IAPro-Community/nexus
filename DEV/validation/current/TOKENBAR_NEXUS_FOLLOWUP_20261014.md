# Revisão TokenBar → Nexus — follow-up 2026-10-14

Branch canônica: release/v1.0.0 (implementação direta).

Concluído em código:
- P0 quota pace no monitor preexistente, amostras SQLite 0029, contrato/API,
  Usage/HUD e CSV; a previsão é consultiva e não entra no scheduler.
- P1 leitura externa opt-in do Claude, JSONL incremental com offset de linha
  completa, deduplicação de mensagens e tabelas isoladas (migração 0030).
- P2 diagnóstico público OpenAI/Claude/GitHub com cache e falhas isoladas.
- Interfaces Cost Intelligence e Capacity Monitor expõem recursos novos.
- Testes de store/scanner/HTTP adicionados, mas não executados localmente.

Invariantes de segurança:
- Quota desconhecida nunca vira quota cheia ou grátis.
- Nenhuma conta é inferida de transcripts sem verificação de identidade.
- Uso externo nunca se soma ao ledger de custos medidos ou estimados.
- Dados de prompt e credenciais não são salvos.
- Nenhuma implementação TokenBar GPL foi copiada.
- Sem alterações PTY/ConPTY, scroll, input Codex, túnel ou autenticação nativa.

Pendente: executar go test e quality:full, E2E no navegador, smoke Windows,
e validar o mesmo SHA em Linux/Windows/macOS. O Actions do GitHub pode ficar
queued/cancelled por créditos; não considerar um run cancelado como aprovação.

## Checkpoint de estabilização (2026-10-15)

- Forecast aceita somente quota LIVE de origem Official API, CLI output ou
  Response Headers; inferência LOCAL_FILES/OBSERVATION é rejeitada.
- Importação manual e exclusão de metadados externos são mutuamente exclusivas
  (HTTP 409 na concorrência); cada operação conserva consentimento/CSRF.
- Sanitização de tamanho e tempo de metadados do JSONL; correção de posição
  da seção de status público e de estado indisponível.
- Scripts PowerShell e Bash de certificação por commit exato adicionados.
- Verificado no GitHub: migration 0029 e 0030 presentes, release com
  avanço fast-forward e Actions sem conclusão verde. Qualidade não executada
  no ambiente local; não declarar release certificada.

## Certificação incremental — 2026-10-15

- A exclusão pela UI deixou de depender da contagem de mensagens dos últimos
  7 dias: registro antigo continua apagável; teste HTTP valida isso.
- O script `--browser` agora executa E2E HTTP + Playwright para responsividade,
  acessibilidade e temas. O teste Capacity isolado é HTTP, não browser.
- Added CSV auth UI Vitest regressions and shared CSV formula safety tests.
- Go/browser CI final green ainda **não obtido** no momento desta edição.
