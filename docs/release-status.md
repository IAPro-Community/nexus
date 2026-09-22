# Status público da release do Nexus v1

Atualizado em **22 de setembro de 2026**.

| Campo | Estado |
| --- | --- |
| Programa | fechamento do Nexus 1.0 |
| Versão reportada pelo source | `1.0.0-beta.4` |
| Última beta publicada registrada | `v1.0.0-beta.1` |
| Release estável v1 | **não autorizada** |
| Veredito atual | **`NEXUS_1_0_CANDIDATE = NO_GO`** |

## O que esse status significa

O source candidato contém a implementação de grande parte do contrato v1, mas
**implementação funcional não é a mesma coisa que certificação de release**.

Uma página de produto pode documentar um recurso presente no source sem afirmar
que a release estável está publicada ou que todas as plataformas estão
certificadas.

## Baseline conhecida

A baseline integrada pelo PR #3 possui evidência verde para os gates registrados
na campanha correspondente, incluindo frontend, Linux/race, segurança, browser,
Desktop Linux, snapshot e smoke DEB.

Essa evidência histórica não certifica automaticamente o head atual nem uma
release futura.

## Plataformas

- Linux possui evidência nativa em campanhas anteriores da linha de fechamento.
- Windows e macOS exigem execução nativa real do candidato final.
- Cross-compilation não substitui runtime nativo.
- Job que falha antes de executar qualquer step deixa a plataforma
  **UNVERIFIED**; não é PASS e também não demonstra sozinho regressão funcional.

## Contrato para stable

Antes de uma v1 estável, o candidato final deve satisfazer os gates definidos
pelo programa de release, incluindo conforme aplicável:

- qualidade, testes e race;
- segurança;
- browser/UI/UX;
- execução nativa nas plataformas declaradas;
- Desktop;
- install/upgrade/uninstall;
- update e cadeia de integridade;
- artifacts/checksums/signing/notarização conforme promessa;
- provider smoke autenticado;
- Mission smoke até `COMPLETED_VERIFIED`;
- soak/dogfooding;
- recovery/restart;
- zero findings BLOCKER/HIGH relevantes.

O repositório também possui um boundary executável de certificação estável para
provar candidate SHA, providers, Mission e soak antes de uma publicação
não-prerelease.

## Como ler a documentação

- **Disponível**: implementação/contrato está presente.
- **Implementado no source candidato**: existe no fechamento v1, mas release
  ainda não está autorizada.
- **Certificado**: requer evidence específica da revisão/plataforma.
- **UNKNOWN / UNVERIFIED**: não há prova suficiente.

## Regra

Até que a certificação final altere explicitamente o status:

**não anunciar o Nexus v1 como GA/stable.**

O ledger interno de engenharia e release permanece preservado no repositório
para maintainers e auditoria, sem ser necessário ao onboarding público.
