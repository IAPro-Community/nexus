# Cobertura documental — Nexus 1.0

> Revisão estrutural de 20/10/2026; adendo de rastreabilidade em 22/10/2026. A cobertura é **mapeamento de arquivos**, não certificação de comportamento ou segurança. A antiga medição de setembro foi substituída pela fonte auditável [features.json](../reference/features.json).

## Fontes de verdade

| Assunto | Contrato |
| --- | --- |
| Funcionalidades, implementações, testes e docs | [Catálogo JSON](../reference/features.json) |
| Matriz pública gerada | [Matriz de funcionalidades](../reference/feature-matrix.md) |
| Instalação e primeiro uso | [Instalação](../getting-started/installation.md), [Quick Start](../quick-start.md) |
| Plataforma e evidências nativas | [Suporte de plataformas](../operations/platform-support.md) |
| Status de release | [Release status](../release-status.md) e evidências do SHA |
| Arquitetura atual | [Architecture](../../ARCHITECTURE.md) e [índice](../architecture/overview.md) |
| Segurança e produto | [Security](../security.pt-BR.md) e [produto](../product/how-it-works.md) |

## Cobertura verificável

O catálogo exige, **para cada recurso registrado**: identificador estável, superfícies, status do source, path da implementação, paths dos testes, link para documento canônico, pré-requisitos, limites e estados de evidência/certificação.

```bash
node scripts/verify-feature-catalog.mjs
node scripts/verify-feature-catalog.mjs --write
node scripts/docs-verify.mjs
```

O verificador de docs executa também o gate do catálogo. `--write` regenera apenas o bloco da matriz, nunca afirma `PASS` de teste/plataforma. Adições funcionais sem entrada de catálogo são defeitos de processo e devem aparecer no code review; detecção automática de novas features não é perfeita só com filesystem.

O catálogo v1 passou de **30 para 36** entradas rastreáveis, incluindo seis
capacidades da reconciliação beta.10: scroll Web, entrada PTY/ConPTY, revisão
explícita com STICKY_LEAD, continuidade por quota, ExecutionTarget/ModelCatalog
e entrega de assets no Desktop. A matriz pública foi regenerada a partir do JSON.
Esses registros continuam com `SOURCE_TEST_PRESENT_NOT_RUN` e
`PENDING_EXACT_SHA`; não são PASS.

## Lacunas técnicas e gates externos

1. A matriz prova que há caminhos de código e testes; **não** prova que os testes executaram ou passaram.
2. O manifesto das capturas aponta para um SHA anterior a alterações recentes de UI. O verificador existente já falha quando há mudança em paths visuais; regenerar capturas em ambiente sintético e no SHA congelado.
3. Arquivos históricos em `DEV/`, `docs/superpowers/`, `docs/validation/` e similares **não são** onboarding, suporte atual ou evidência de release.
4. Releases públicas exigem auditoria do histórico, SBOM/licenças, certificação do mesmo SHA, instaladores/assinaturas e publicação real dos artefatos.
5. CI cancelado, pular job, cross-build ou teste unitário isolado não se transforma em certificação.

## Regra de manutenção

Ao adicionar/alterar função: atualizar código, teste, documento canônico e catálogo na mesma mudança; regenerar a matriz; rodar os gates documentais e funcionais; incluir evidência exata no registro de release. **Não alterar o estado global de certificação no catálogo até existir um mecanismo externo de ingestão de evidência verificável.**
