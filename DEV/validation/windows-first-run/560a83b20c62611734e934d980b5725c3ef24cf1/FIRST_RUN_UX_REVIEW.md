# Revisão de UX/UI da Primeira Execução — Nexus 1.0.0 (Windows)

- **Commit SHA:** `560a83b20c62611734e934d980b5725c3ef24cf1`
- **Público Avaliado:** Novo usuário no Windows 11 sem conhecimento prévio da arquitetura interna do Nexus.

---

## 1. Respostas Objetivas às 8 Perguntas da Homologação

### 1. É evidente o que fazer após instalar?
**Sim.** Com as mudanças introduzidas no commit `560a83b2` (especialmente `firstRunReadiness.ts` e a reestruturação do `ProjectHub`), o Nexus apresenta uma tela inicial clara de acolhimento. Ao abrir o aplicativo, o usuário é direcionado para:
1. Criar ou selecionar um Projeto;
2. Ou vincular um Provedor de IA.
Diferente das versões anteriores, a falta de provedores não bloqueia mais o uso do workspace: o usuário pode criar um projeto e abrir o terminal integrado nativo (Project Shell) imediatamente.

### 2. O Nexus identifica sozinho o que está pronto?
**Sim.** O backend possui descoberta ativa via `internal/doctor` e `internal/app/providers.go`. Na primeira inicialização, ele detecta autonomamente:
- CLIs já instaladas no PATH (Codex, AGY, Gemini, OpenCode);
- Presença e versão do runtime Microsoft Edge WebView2;
- Suporte nativo a ConPTY no Windows;
- Estado de autenticação existente no diretório de perfil (`auth.json` ou tokens locais).
O usuário não precisa cadastrar manualmente ferramentas que o sistema já encontra no PATH.

### 3. O que ainda exige configuração manual?
1. **Autenticação de Provedores:** Caso o usuário deseje usar um provedor ainda não autenticado, é necessário iniciar o fluxo oficial de login (`nexus login <provedor>` ou login via UI). O Nexus não pode e não deve tentar adivinhar ou inventar credenciais.
2. **Instalação de CLIs externas não presentes:** Caso queira usar Claude Code ou Cursor e não os possua na máquina, o usuário deve instalá-los individualmente.

### 4. Existe alguma tela técnica desnecessária?
**Não no fluxo primário.** O modal de boas-vindas e o modal de cadastro de projeto são limpos, contendo apenas o nome do projeto e o seletor de pasta.
Telas avançadas como o detalhamento de rotas IPC, tabelas de cota profunda e auditoria de segurança (`nexus security`) ficam devidamente resguardadas em abas dedicadas de Configurações e Diagnóstico, acessíveis somente quando o usuário explicitamente as solicita.

### 5. O usuário consegue identificar e resolver um problema?
**Sim.** As mensagens de erro e diagnósticos exibem instruções operacionais diretas. Por exemplo:
- Provedor ausente: *"binary not found in PATH"*, orientando a instalação.
- Falta de autenticação: *"AUTH_REQUIRED"*, indicando o comando de login necessário.
- Erro no terminal: as falhas de conexão ConPTY reportam o código de erro Win32 real sem mascarar com telas brancas genéricas.

### 6. O diagnóstico exibe informações verdadeiras?
**Sim, com rigor exemplar.** O Nexus não inventa dados quando não tem conectividade ou credenciais:
- Se uma conta não estiver logada, a cota e o consumo aparecem como `UNKNOWN`, com fonte `NONE` e custo `0`.
- Não há cobranças fictícias ou estimativas arbitrárias de tokens.
- O diagnóstico do `nexus doctor` inspeciona permissões reais de arquivo e integridade do banco SQLite.

### 7. A interface orienta sem interromper o trabalho?
**Sim.** O HUD fica recolhido na parte inferior direita com design não-obstrutivo (suporta modo compacto/mini), exibindo o status de prontidão sem sobrepor a área de código. As notificações do sistema e atenção de agentes operam de forma assíncrona.

### 8. É possível começar a usar o produto com poucos cliques?
**Sim.** O fluxo típico no Windows consiste em:
1. Clicar em "Novo Projeto";
2. Clicar no botão de navegação de pastas (que abre o seletor de diretórios nativo do Windows);
3. Confirmar a pasta escolhida (o nome do projeto é preenchido automaticamente com o nome da pasta);
4. O workspace é aberto imediatamente, liberando o Project Shell e as ferramentas de automação.

---

## 2. Avaliação de Mudanças Específicas do Commit `560a83b2`

| Recurso | Avaliação UX | Observações |
| :--- | :--- | :--- |
| **`firstRunReadiness.ts`** | **Excelente** | Desacopla a obrigatoriedade de IA do cadastro de projetos. Permite onboarding fluido mesmo sem nenhuma conta de IA cadastrada. |
| **Placeholders de Caminho** | **Excelente** | Substituiu o placeholder legado do Unix (`/projetos/saas-app`) pelo caminho canônico Windows (`C:\Projetos\meu-repositorio`). |
| **`AddProjectModal.tsx`** | **Muito Bom** | Inspeciona em tempo real se o diretório selecionado contém repositório Git e sugere o nome da pasta de forma reativa. |
| **Terminal ConPTY Input** | **Excelente** | A resolução do problema histórico de fragmentação ESC permite que menus interativos do Codex e navegação com setas funcionem com naturalidade nativa. |
