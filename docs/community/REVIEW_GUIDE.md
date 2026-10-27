# Guia de triagem e revisão comunitária do Nexus

A manutenção do upstream precisa ser acolhedora **sem comprometer a segurança** de um runtime que executa agentes com acesso a arquivos, terminal, navegador e credenciais.

## Propostas e issues

- Verificar duplicatas, problema do usuário, ambiente, passos de reprodução, risco e critério de aceite.
- Responder com decisão ou próximo passo; indicar publicamente quando o problema já foi corrigido e qual commit/versão o resolveu.
- Para mudanças grandes, usar proposta de design/ADR antes do PR, incluindo alternativas e impacto em compatibilidade.
- Não pedir evidência sensível em issue. Vulnerabilidades: [SECURITY](../../SECURITY.md).
- Nomear pequenas tarefas independentes e documentos para iniciantes; não prometer prazo de resposta sem capacidade operacional.

## PRs de código (após clearance jurídico e licença open source)

- Verificar autorização do contribuinte (DCO após ativação), compatibilidade de licença e autoria de dependências/assets.
- Confirmar escopo, teste de regressão, i18n, acessibilidade, documentação, linters e integridade de mudanças em branches.
- Proibir novos schedulers/runtimes/armazenamentos de memória paralelos. Respeitar as autoridades existentes do Nexus e a independência do Maestro.
- Alterações em permissões, execução de comandos, login, terminais ou browsers devem ter revisão de segurança adicional.
- Testes verdes no SHA de PR não equivalem a certificação de release; release precisa de homologação por plataforma, assinatura e evidência do SHA final.
- Não fazer merge por mera contagem de aprovações. Mantenedores documentam trade-offs, riscos e a decisão.

## Aprovação e conflitos

Veja [GOVERNANCE](../../GOVERNANCE.md) para responsáveis, regras de revisão e resolução de impasses. Contribuições permanecem do autor, salvo instrumento próprio válido; nunca presumir cessão de direitos. Relatórios de autoria e segurança são tratados privadamente quando necessário.
