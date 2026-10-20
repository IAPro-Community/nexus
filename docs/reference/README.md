# Catálogo funcional rastreável

[features.json](features.json) descreve 30 capacidades do source com identificador, superfícies, status, arquivo de implementação, arquivo de teste, documento canônico, requisitos, limites e estados de evidência. A [matriz](feature-matrix.md) é gerada do JSON, não editada manualmente.

```bash
node scripts/verify-feature-catalog.mjs --write
node scripts/verify-feature-catalog.mjs
node scripts/docs-verify.mjs
```

`implemented` significa presença no código, `experimental` exige validação integrada, `optional` é uma integração dispensável e `internal` não é contrato de API pública. `SOURCE_TEST_PRESENT_NOT_RUN` **não** significa testes PASS. `PENDING_EXACT_SHA` **não** significa plataforma suportada/certificada.

O verificador exige paths válidos e tabela sincronizada, mas não detecta automaticamente uma funcionalidade nova que não tenha sido cadastrada: code review deve exigir alteração de catálogo, implementação, testes e guia na mesma mudança. A fonte de certificação permanece em [release](../operations/release.md).
