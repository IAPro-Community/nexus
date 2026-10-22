# Checklist de exposição pública e publicação estável — Nexus v1

> **Observado em 20/10/2026:** o repositório `IAPro-Community/nexus` está **PUBLIC** no GitHub. A varredura automatizada dos refs disponíveis foi executada em 20/10/2026: 82 detecções potenciais no histórico e 17 no HEAD do snapshot auditado (não classificadas; podem incluir duplicações e falsos positivos). **A classificação manual e a remediação permanecem pendentes.** Isso configura uma **exposição pública a avaliar**, não prova de credenciais vazadas. Tornar o repositório privado, se decidido, não remove clones já obtidos. A v1 estável e os binários permanecem **NO-GO**.

## Resposta imediata à abertura já realizada

- [ ] Confirmar quem aprovou a mudança de visibilidade e o inventário de branches/tags/commits que ficaram acessíveis.
- [x] Executar varredura automatizada nos refs obtidos pelo workflow público, usando Gitleaks v8.24.2, sem publicar resultados brutos.
- [ ] Classificar privadamente as detecções (82 no histórico / 17 no HEAD auditado), verificar falsos positivos, dados pessoais e os materiais de `DEV/`, `TODO/` e históricos. Não presumir que cada detecção é uma credencial ativa.
- [ ] Em caso de segredo confirmado, revogar/rotacionar **na origem**, verificar logs de utilização indevida e seguir resposta a incidentes. Remover texto do HEAD não é remediação de histórico.
- [ ] Definir destino do histórico interno e política pública de branches; não reescrever refs compartilhadas sem coordenação e plano de migração.
- [ ] Habilitar canal privado de vulnerability reporting e regras de branch/proteções apropriadas, de acordo com as permissões administrativas do repositório.

## Revisão local privativa das detecções

Não publicar relatórios brutos em artifacts do GitHub Actions ou issues públicas. Em um checkout completo e limpo, com Gitleaks v8.24.2 instalado, configure uma pasta privada **fora** do repositório:

```bash
git fetch --all --tags --prune
mkdir -p "$HOME/.local/state/nexus-security"
chmod 700 "$HOME/.local/state/nexus-security"
NEXUS_PRIVATE_AUDIT_DIR="$HOME/.local/state/nexus-security" bash scripts/audit-publication.sh
```

O script interrompe a aprovação quando encontra suspeitas e mantém relatório redigido fora do Git. Revisar os registros localmente, sem transmitir `Secret`, `Match`, trechos de código ou caminhos privados a chats/tickets públicos. Classificar por provedor, validade, escopo, ambiente e prioridade; revogar ou rotacionar credenciais reais ainda ativas na origem. Documentar o encerramento sem publicar os valores.

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
- [x] Iniciar catálogo verificável de recursos, testes e documentos (não equivalente a suite executada).
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

## Estratégia de saneamento pós-abertura

1. Registrar a exposição pública já existente e a decisão formal de visibilidade daqui em diante.
2. Auditar refs e artefatos, remediar vazamentos eventuais e avaliar repositório público separado com histórico saneado **sem apagar evidência nem destruir branches**.
3. Atualizar notices/SBOM, screenshots e matrizes; manter issues e security advisories sob políticas claras.
4. Publicar anúncio como **source-available**, não “open source” OSI.
5. Promover binários `v1.0.0` somente com veredito GO no SHA imutável da candidata.

**Observação:** mesmo snapshots públicos podem ser clonados; licença não equivale a DRM.
