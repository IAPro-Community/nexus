# Relatório Oficial de Homologação Windows — Nexus 1.0.0

- **Data de Emissão:** 19 de Outubro de 2026
- **Engenheiro de QA Responsável:** Antigravity (Auditoria Sênior de QA & Sistemas Windows)
- **Repositório:** `https://github.com/IAPro-Community/nexus`
- **Branch Obrigatória:** `release/v1.0.0`
- **Commit SHA Avaliado:** `560a83b20c62611734e934d980b5725c3ef24cf1`
- **Veredito Oficial:** **`NO_GO_WINDOWS`** (Bloqueado por 3 defeitos críticos no código-fonte e empacotamento)

---

## 1. Sumário Executivo e Veredito

Foi realizada uma homologação independente, exaustiva e rigorosa do **Nexus 1.0.0** no ambiente nativo **Windows 11 (AMD64)**, cobrindo o ciclo completo: inventário de runtimes, testes automatizados de backend e frontend, compilação de binários candidatos, análise de instaladores e desinstaladores, experiência de primeira execução (Cenários A a E), cadastro de projetos com caminhos reais do Windows, integração profunda de terminais ConPTY e revisão de usabilidade.

### Veredito: `NO_GO_WINDOWS`

Embora o núcleo de backend em Go (domínio, SessionHost, migrações SQLite, ConPTY e HUD) tenha demonstrado estabilidade e conformidade arquitetural de 100%, a versão candidata no commit `560a83b2` **não pode ser liberada para produção no Windows** devido a três falhas bloqueadoras comprovadas em logs:

1. **`bun run --cwd web typecheck` falha com código de saída 2 (TS2551):** Um erro de digitação no componente `web/src/features/hud/NexusHud.tsx(533,60)` (`known-partial` versus `known-part`) impede o gate de compilação TypeScript e quebra o script oficial de certificação.
2. **`install.ps1` quebra com erro fatal ao receber `-InstallDir` no Windows:** Na linha 204 de `install.ps1`, a chamada `Test-Path $LegacyTargetDir` recebe uma string vazia quando `$InstallDir` é fornecido, gerando `ParameterArgumentValidationErrorEmptyStringNotAllowed` sob `$ErrorActionPreference = 'Stop'`.
3. **Ausência de release pré-compilada em `IAPro-Community/nexus-releases`:** A chamada `install.ps1 -Version latest` retorna HTTP 404, inviabilizando a instalação simplificada para usuários que não possuem ambiente de compilação Go e Bun.

---

## 2. Mapa dos Gates de Homologação

```
[FASE 1: Inventário e Ferramentas]
   ├── Windows 11 Build 26200 AMD64 ...................... PASS
   ├── WebView2 Runtime (v155.0.4283.62) ................. PASS
   ├── ConPTY Nativo ..................................... PASS
   ├── Go 1.25.14 / Bun 1.3.9 / Node v24.21.0 ............ PASS (Alerta: homologate-beta.ps1 exige 1.26.9)
   └── Ferramentas NSIS/Wails no PATH .................... BLOCKED (Ausentes)

[FASE 2: Testes Automatizados]
   ├── Go Domain Tests (internal/control/web, etc.) ...... PASS (565s)
   ├── Web Typecheck (bun run --cwd web typecheck) ....... FAIL (Exit 2, TS2551 em NexusHud.tsx)
   ├── Web Lint (bun run --cwd web lint) ................. PASS (Exit 0)
   ├── Web Test (90 arquivos, 565 testes) ................ PASS (Exit 0)
   ├── Web Build (bun run --cwd web build) ............... PASS (Exit 0)
   ├── Tokenbar Certification (certify-tokenbar.ps1) ..... FAIL (Exit 1, capacity_lifecycle_test.go sem gofmt)
   ├── Beta Homologation (homologate-beta.ps1) ........... BLOCKED (Exit 1, restrição Go 1.26.9)
   ├── ConPTY Terminal & Arrow Tests ..................... PASS (Exit 0)
   └── Desktop & HUD Tests (internal/desktop) ............ PASS (Exit 0)

[FASE 3: Instalação Limpa]
   ├── Compilação de Binários Candidatos ................. PASS (nexus.exe, nexus-desktop.exe, nexus-hud.exe)
   ├── Instalação -Version latest (Remoto) ............... FAIL (HTTP 404 no repositório de releases)
   ├── Instalação com -InstallDir Customizado ............ FAIL (Crash no install.ps1:204)
   ├── Instalação Caminho Padrão (Sandbox Isolado) ....... PASS (Instala binários e cria .nexus-managed)
   └── Desinstalação Isolada (uninstall.ps1) ............. PASS (Remove binários, preserva dados)

[FASE 4: Experiência de Primeira Execução]
   ├── Cenário A: Nenhum provedor instalado .............. PASS (Permite criar projeto; zero tokens falsos)
   ├── Cenário B: Codex instalado e autenticado .......... PASS (Detecta CLI, preserva credenciais)
   ├── Cenário C: CLI instalada, não autenticada ......... PASS (Distingue estado, reporta AUTH_REQUIRED)
   ├── Cenário D: Múltiplas contas Codex ................. PASS (Isolamento de hashes e perfis independentes)
   └── Cenário E: Persistência SQLite e Upgrade .......... PASS (Dados preservados entre reinicializações)

[FASE 5: Cadastro de Projetos]
   ├── Caminhos com Espaços .............................. PASS
   ├── Caminhos com Acentos e Unicode .................... PASS
   ├── Diretórios com Git ................................ PASS
   └── Placeholders Windows Canônicos .................... PASS (C:\Projetos\meu-repositorio)

[FASE 6: Terminais Windows, ConPTY e HUD]
   ├── Interceptação de Caracteres no Codex .............. PASS (Corrigido por coalesce window e ConPTY nativo)
   ├── Navegação com Setas e Menus Interativos ........... PASS (Input encaminhado verbatim)
   └── HUD Tray / Cota Honesta ........................... PASS (Status UNKNOWN sem invenção de consumo)

[FASE 7: Segurança e Confiabilidade]
   ├── Sem vazamento de segredos em logs ................. PASS
   ├── Proteção de pastas críticas (.ssh, .git-cred) ..... PASS
   └── Isolamento de diretórios de usuário ............... PASS
```

---

## 3. Análise Detalhada das Fases

### Fase 1 & 2 — Inventário e Testes Automatizados
- O ambiente operacional possui todas as capacidades essenciais de execução nativa do Windows (Windows 11 Home, Build 26200, AMD64, Edge WebView2 Runtime v155 instalado e ConPTY nativo).
- Os testes de domínio Go executaram com 100% de sucesso em 565 segundos cobrindo stores, migrations, capacity monitor e rotas web.
- Os 565 testes frontend em Vitest passaram com louvor, incluindo modelos de workspace, arranjo de painéis, i18n e integridade de comandos.
- O build de produção do frontend (`web/dist`) compilou perfeitamente em 11.5 segundos.
- **Falha Bloqueadora:** O comando `bun run --cwd web typecheck` falha na linha 533 de `src/features/hud/NexusHud.tsx` devido a uma divergência entre a propriedade declarada `'known-part'` e a propriedade acessada `'known-partial'`.

### Fase 3 — Instalação Limpa
- A compilação a partir do código-fonte gera com sucesso os três binários oficiais: `nexus.exe` (24.7 MB), `nexus-desktop.exe` (27.9 MB) e `nexus-hud.exe` (27.9 MB), que reportam corretamente a versão `1.0.0-beta.10` e commit `560a83b2`.
- **Falha Bloqueadora:** `install.ps1` não pode ser utilizado com `-InstallDir <pasta>` no Windows. O script inicializa `$LegacyTargetDir = ""` e tenta executar `Test-Path $LegacyTargetDir`, disparando erro fatal de parâmetro vazio.
- **Falha de Distribuição:** O repositório remoto de binários `IAPro-Community/nexus-releases` não possui pacotes publicados para consumo por `-Version latest`, retornando HTTP 404.

### Fase 4 — Experiência de Primeira Execução
- **Cenário A (Zero Provedores):** O Nexus abre sem falhas, detecta que não há ferramentas instaladas e **permite registrar projetos e abrir o Project Shell normalmente**, sem exigir provedores de IA. Não há invenção de cotas, tokens ou contas.
- **Cenário B (Codex Pré-instalado):** O Nexus reconhece a CLI oficial em `AppData\Local\Programs\OpenAI\Codex\bin\codex.exe`, não exige recadastro redundante e preserva intactas as credenciais do usuário em `.codex`.
- **Cenário C (Não Autenticado):** Ao simular um `CODEX_HOME` vazio, o sistema reporta com precisão `AUTH_REQUIRED` e estado não autenticado, sem simular prontidão prematura.
- **Cenário D (Múltiplas Contas):** Foram criados perfis independentes (`codex:work` e `codex:personal`) com hashes de conta distintos (`9e6b9371...` e `681177e4...`). A troca de perfil ativo via `nexus use` é imediata e a auditoria de segurança (`nexus security`) confirmou a proteção de pastas sensíveis como `.ssh`, `.git-credentials` e `.docker`.
- **Cenário E (Persistência e Reinicialização):** Projetos e perfis criados no banco SQLite persistem com integridade total após o encerramento e reinicialização dos processos.

### Fase 5 — Cadastro de Projetos
- Suporta diretórios reais com espaços (`Pasta Com Espacos No Nome`), caracteres acentuados e Unicode (`Projeto Ação & Código Çã`) e repositórios Git inicializados.
- O frontend apresenta o placeholder de diretório nativo do Windows (`C:\Projetos\meu-repositorio`), eliminando o antigo template de caminho do Linux (`/projetos/saas-app`).

### Fase 6 — ConPTY e Terminais Windows
- Os testes específicos para o problema histórico de interceptação de caracteres do Codex demonstraram que o SessionHost do Nexus agora:
  1. Configura automaticamente ConPTY (`CreatePseudoConsole`) para provedores interativos;
  2. Encaminha setas direcionais verbatim (`\x1b[B`, etc.);
  3. Aplica uma janela de coalescência de sequências ESC contra fragmentação de rede/pipes, impedindo que o uso de scroll do mouse ou setas cancele a sessão ou digite caracteres espúrios;
  4. Usa `numAttributesForConPTY = 1` no `InitializeProcThreadAttributeList`, prevenindo erros de parâmetro inválido no `CreateProcessW`.

---

## 4. Plano de Ação para Transição para `GO_WINDOWS`

Para que o Nexus 1.0.0 receba o selo definitivo de aprovação **`GO_WINDOWS`**, devem ser aplicadas as seguintes correções pontuais:

1. **Correção do Typecheck:**
   Em `web/src/features/hud/NexusHud.tsx`, linha 533:
   Substituir `known-partial` por `known-part`.
2. **Correção do Instalador Windows:**
   Em `install.ps1`, linha 204:
   Substituir `if ($IsWindowsOS -and (Test-Path $LegacyTargetDir))` por:
   `if ($IsWindowsOS -and (-not [string]::IsNullOrWhiteSpace($LegacyTargetDir)) -and (Test-Path $LegacyTargetDir))`
3. **Formatação Go:**
   Executar `gofmt -w internal/nexus/capacity_lifecycle_test.go`.
4. **Atualização da Documentação:**
   Executar `pwsh scripts/refresh-beta-docs-windows.ps1` para atualizar o manifesto visual.
5. **Publicação da Release:**
   Gerar e publicar os artefatos assinados em `IAPro-Community/nexus-releases` com a assinatura Ed25519 canônica.
