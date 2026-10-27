# Nexus — auditoria de preparação open source (27/10/2026)

**Branch:** `release/v1.0.0`  
**Tipo:** auditoria documental, jurídica preliminar e revisão estática dos workflows.  
**Resultado:** PREPARED / **NOT_RELICENSED** / **NO_GO** para anunciar licença open source ou release estável.

## Fatos verificados via arquivos do repositório

- `LICENSE` ainda contém PolyForm Strict 1.0.0 (source-available, não open source OSI).
- `VERSION` e `web/package.json` declaram `1.0.0-beta.11` em 27/10/2026; docs principais foram corrigidos para essa verdade, preservando relatos históricos beta.10.
- `docs/legal/LICENSING.md`, `docs/legal/THIRD_PARTY_NOTICES.md` e `docs/PUBLICATION_CHECKLIST.md` documentam titulares e terceiros ainda pendentes de confirmação.
- O checklist histórico registra 82 alertas potenciais em refs/histórico e 17 no HEAD daquela auditoria de 20/10; **isso não comprova vazamento** e não é uma rechecagem do HEAD de 27/10. Classificação privada e eventual rotação continuam bloqueadoras.
- O workflow `.github/workflows/ci.yml` era disparado por `pull_request` e executava múltiplos jobs self-hosted (incluindo Linux/Windows/macOS). A alteração de código removeu esse gatilho.
- `.github/workflows/community-pr.yml` foi criado para PRs em runner GitHub-hosted efêmero, token `contents: read`, sem secrets ou ambientes de assinatura. Há checks para frontend, Go, documentação e higiene do diff.

## Alterações aplicadas nesta revisão

- Documentação de transição responsável para Apache-2.0: `docs/legal/OPEN_SOURCE_TRANSITION.md`. A licença-alvo **não** substitui `LICENSE` sem autorização.
- Política de marca e forks: `docs/legal/TRADEMARK_POLICY.md`, `docs/community/FORKS_AND_UPSTREAM.md`.
- Contribuições/governança/revisão: `CONTRIBUTING.md`, `GOVERNANCE.md`, `docs/community/REVIEW_GUIDE.md`, `.github/PULL_REQUEST_TEMPLATE.md`, `.github/ISSUE_TEMPLATE/feature_request.yml`.
- Sustentabilidade e onboarding: `docs/community/FUNDING.md`, `docs/community/README.md`, `docs/community/ADMIN_SETUP.md`. **Nenhum link de pagamento fictício foi criado.**
- Documentação pública alinhada: README PT/EN/ES, TLDR, FAQ, roadmap, índice de docs, STYLE_GUIDE, PRIVACY, SECURITY, licença vigente, notices de terceiros, publicação e release status.
- `internal/release/installer_test.go`: guarda que README mencione PolyForm como vigente e Apache-2.0 como alvo não vigente, em vez de rejeitar qualquer menção a Apache.

## Lacunas que exigem ação do titular/admin

- [ ] Direitos patrimoniais dos arquivos/contribuições/assets e autorização específica para relicenciar, ou substituição de partes não comprovadas.
- [ ] Conclusão da triagem privada de segredos do histórico e direitos de terceiros (com SBOM exato).
- [ ] Publicação efetiva do `LICENSE` oficial Apache-2.0 e `NOTICE` após a liberação jurídica; atualização conjunta dos metadados e teste de licença.
- [ ] Ativação e teste de rulesets/required checks de `main` e `release/v1.0.0`, approvals e controle de bypass.
- [ ] Ativação de private vulnerability reporting e revisão de acessos/segredos de runners históricos.
- [ ] Configuração de GitHub Sponsors e `.github/FUNDING.yml` somente depois de verificar conta apta.
- [ ] Execução real no SHA final: `node scripts/docs-verify.mjs`, `go test ./internal/release/... ./...`, `bun run quality:full`, validação dos dois workflows, varredura de secrets fora do repositório e homologação de release.

## Evidência da revisão (limitações)

Arquivos e commits foram lidos/escritos via GitHub connector. Não houve checkout local completo, nem `go test`, `bun`, validação YAML por um runner, execução de GitHub Actions ou verificação de settings administrativas. Foram verificados via GitHub todos os **16 caminhos relativos distintos** referenciados pelos nove primeiros documentos comunitários/legais revisados; isso não substitui `docs-verify` completo. Não afirmar CI verde ou release publicada.

## Critério de pronto

**PRODUTO OPEN SOURCE:** matriz de direitos fechada + licença OSI efetiva no commit + contributors habilitados sob termos válidos + política de marca/terceiros correta.  
**SEGURANÇA DE COLABORAÇÃO:** fork PR executa apenas em runner isolado, rulesets ativos, contas privilegiadas/segredos auditados.  
**RELEASE:** evidência de teste/build/assinaturas e plataformas no exato SHA.

Até esses três gates serem aprovados, não anunciar o Nexus como já licenciado Apache-2.0. Documentação de intenção não é outorga de direitos.
