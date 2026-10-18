# Checklist de publicação pública — Nexus v1

**Estado inicial (18/10/2026): BLOQUEADO.** O repositório estava **privado** e o source da `release/v1.0.0` declarava `1.0.0-beta.10`. Este checklist **não autoriza abrir o repositório nem publicar binários**.

## Gate P0 — direitos e informações expostas

- [ ] Confirmar titularidade/autorização de relicenciamento do código, contribuições, documentação, marca e assets.
- [ ] Revisão jurídica da PolyForm Strict, termos beta anteriores e política de uso comercial.
- [ ] Auditar **todas as branches, tags, commits, arquivos e histórico**, não só HEAD; procurar credenciais, tokens, `.env`, `DEV/`, logs, dumps, dados pessoais, chaves de assinatura, URIs privadas e projetos de clientes.
- [ ] Decidir se `DEV/`, `TODO/`, prompts internos, relatórios de homologação e fixtures sensíveis devem ir para um repositório privado separado.
- [ ] Se houver dados sensíveis no histórico, rotacionar credenciais e criar distribuição limpa com histórico controlado. Apagar um arquivo do HEAD não o apaga dos commits antigos.
- [ ] Revisar dependências, SBOM, notices, licença de fonts/assets, GPL/AGPL e atribuições antes de disponibilizar código e binários.
- [ ] Revisar `.github/workflows`, permissions, actions e qualquer job capaz de alcançar secrets depois da abertura.

## Gate P0 — verdade documental

- [x] Selecionar modelo source-available não comercial, sem redistribuição/modificação (PolyForm Strict).
- [x] Criar guia legal e TL;DR; alinhar README e guia rápido ao status do source, sem declarar stable.
- [ ] Confirmar documentação de versão, canais publicados, links de instalação e suporte por plataforma com inventário remoto validado.
- [ ] Conferir dados de contato de suporte e política de privacidade contra a implementação real.
- [ ] Executar `node scripts/docs-verify.mjs` e `git diff --check` no SHA exato da publicação, salvando evidência.
- [ ] Validar README, instalação e primeiro uso em instalação limpa Linux, Windows e macOS, declarando somente os targets certificados.

## Gate P0 — release de software

- [ ] Congelar SHA de candidata e executar Go/tests/race/vet/lint, frontend e browser E2E no mesmo SHA.
- [ ] Validar no hardware: Codex/TUI/scroll/PTY/ConPTY, autenticação/túnel, Desktop, HUD, atualizador e uninstall.
- [ ] Produzir artefatos com checksums, SBOM, procedência e assinaturas; certificar NSIS/macOS conforme suporte declarado.
- [ ] Verificar `VERSION`, binários, nome da tag e manifest de update contra o mesmo SHA.
- [ ] Documentar `GO` explícito para cada plataforma; falha ou evidência ausente = `NO-GO`.
- [ ] Só após gates aprovados, promover código/publicar tag e distribuição. Código público e release estável são decisões separadas.

## Estratégia recomendada de abertura

1. Manter `release/v1.0.0` canônica e privada enquanto atualiza documentação e testes.
2. Auditar o histórico. Se estiver limpo, abrir o repositório **somente após aprovação explícita**. Caso contrário, publicar um snapshot sanitizado em um repositório de distribuição de source separado; manter histórico interno privado.
3. Inicialmente habilitar **issues e security advisories**; não solicitar PRs de código enquanto a política jurídica para contribuições não estiver formalizada.
4. Publicar anúncio como **source-available**; não usar “open source” sem licença OSI.
5. Publicar `v1.0.0` estável apenas quando a matriz de certificação exata estiver aprovada.

**Observação:** mesmo snapshots públicos podem ser clonados; licença não equivale a DRM.
