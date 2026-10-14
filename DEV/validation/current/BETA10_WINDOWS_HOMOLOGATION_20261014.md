# Homologação Windows beta.10 — 2026-10-14

## Resultado automatizado no SHA exato

O workflow hosted [Windows beta.10 homologation, run 38486183393](https://github.com/IAPro-Community/nexus/actions/runs/38486183393) concluiu **success** na tentativa 2, certificando o source SHA `bdc72325a9409c4b3da3dec94631325f0959ea92` da branch `fix/windows-beta10-homologation-20261008`. Ambiente: Windows Server 2025 AMD64, Go 1.27.2, Bun 1.3.9 e Node 22.14.0.

Gates aprovados:

- Documentação: 178 arquivos Markdown e 33 assets/arquivos requeridos.
- Web: `bun install --frozen-lockfile` e `quality:full`; 84 arquivos de teste e 504 testes passaram.
- Go: formatação, vet, golangci-lint v2.14.0 (0 issues), govulncheck (nenhuma vulnerabilidade), contrato de versão e `go test -count=1 ./...`.
- Regressões focadas e suítes browser E2E, incluindo verificações Axe e visuais.
- Builds Windows finais unsigned de CLI, Desktop e HUD.

Logs: [artifact 11861513083](https://github.com/IAPro-Community/nexus/actions/runs/38486183393/artifacts/11861513083). A primeira tentativa falhou por erro transitório de socket do browser; a segunda passou no mesmo source SHA.

## Integração à release

Na consulta de 2026-10-14, `release/v1.0.0` aponta para `b2a74da2b434eb0dbfe925c383e6d7fc39e7dcc9`; a fonte homologada da candidata está em `bdc72325a9409c4b3da3dec94631325f0959ea92`. O GitHub compara os históricos como divergentes: candidata 171 commits à frente e 71 atrás da release. Não há PR aberto entre essa candidata e `release/v1.0.0`. Portanto o PASS Windows pertence ao candidato beta.10 e ainda não certifica código integrado na release; integração seletiva/revisão e repetição dos gates Windows no SHA integrado continuam obrigatórias.

## Diagnóstico do runner Windows

O gate incluiu `nexus doctor`. No runner limpo, ele registrou avisos: diretório de configuração ainda não criado; Windows Credential Manager com capacidade `UNSUPPORTED`; ConPTY detectado sem smoke nativo pelo doctor; shell Desktop marcado como `SKIPPED` por exigir sessão interativa; e alguns providers opcionais ausentes no PATH. São limites declarados pelo diagnóstico, não falhas ocultadas: nenhum fluxo autenticado de provider nem integração nativa do Credential Manager foi certificado por este run.

## Escopo ainda aberto

A homologação automatizada Windows está **PASS** neste SHA. A tentativa não executou assinatura nem publicou binários de distribuição. Permanecem pendentes os checks manuais de interação física do tray, os fluxos autenticados Provider/Mission/Voice, a assinatura e trust root, o instalador assinado e a integração da candidata beta.10 por PR em `release/v1.0.0`.

**Veredito de distribuição/tag: NO-GO.** Não houve alteração da branch `release/v1.0.0` nem criação de tag. Este relatório trata da candidata beta.10 e não substitui a homologação histórica beta.7.

