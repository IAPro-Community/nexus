# Release e certificação

Release é um processo de **evidência**, não apenas alteração de versão.

Um build local, snapshot, cross-compilation ou versão declarada no source não
prova publicação/certificação.

## Fonte de verdade

Antes de distribuir, consulte:

[`DEV/validation/current/RELEASE_STATUS.md`](../../DEV/validation/current/RELEASE_STATUS.md)

## Gates conceituais

O candidato final deve provar, conforme o escopo declarado:

- qualidade/lint/testes;
- segurança;
- browser/UI/UX;
- runtime nativo nas plataformas suportadas;
- Desktop quando declarado;
- install/upgrade/uninstall;
- update/integridade;
- artifacts e checksums;
- signing/notarização quando prometidos;
- provider smoke;
- Mission smoke;
- dogfooding/soak;
- ausência de blockers conhecidos relevantes.

## Same-SHA

Evidência de uma revisão anterior não certifica automaticamente o candidato
final. O relatório deve identificar exatamente qual SHA foi validado.

CI de pull request pode executar um synthetic merge result; isso prova
mergeabilidade naquele contexto, mas release provenance deve ser associada ao
SHA imutável usado para produzir os artifacts.

## Plataformas

Cross-compilation prova compilação, não execução nativa.

Falha de runner antes de qualquer step deixa a plataforma **UNVERIFIED**; não
deve ser interpretada nem como PASS nem como regressão funcional demonstrada.

Veja [Suporte de plataformas](platform-support.md).

## Distribuição

Formas documentadas:

- scripts: [Instalação nativa](../getting-started/installation.md);
- DEB/RPM/NSIS/winget: [Pacotes](../getting-started/packages.md);
- Docker: [Compatibilidade Docker](../getting-started/docker-compat.md).

Docker não certifica Desktop nativo.

## Maestro

Maestro é opt-in. Não deve ser instalado silenciosamente nem virar dependência
da execução básica do Nexus.

## Stable v1

Somente promova a release estável quando o status autoritativo mudar com
evidência correspondente. Até lá, documentação do source candidato não deve ser
interpretada como anúncio de GA.
