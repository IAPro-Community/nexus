# Release e certificação

Release é um processo de **evidência**, não apenas alteração de versão.

Um build local, snapshot, cross-compilation ou versão declarada no source não
prova publicação/certificação.

## Fonte de verdade

Antes de distribuir, consulte:

[Status público da release](../release-status.md)

## Gates conceituais

O candidato final deve provar, conforme o escopo declarado:

- qualidade/lint/testes;
- segurança;
- browser/UI/UX;
- runtime nativo nas plataformas suportadas;
- Desktop quando declarado;
- install/upgrade/uninstall;
- update/integridade;
- artifacts, hashes e manifesto assinado;
- signing/notarização somente quando efetivamente prometidos/publicados;
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
- DEB e archives de CLI/Desktop: [Pacotes](../getting-started/packages.md);
- Docker: [Compatibilidade Docker](../getting-started/docker-compat.md).

Docker não certifica Desktop nativo.

## Maestro

Na beta atual o Maestro ainda é opcional. A arquitetura aprovada para Nexus
1.0 exige o Maestro como dependência gerenciada e autoridade de contexto/skills/
memória semântica; Maestro permanece independente. A migração deve manter
backup e rollback e não alterar uma instalação externa sem autorização. O
[contrato obrigatório de aceite do Workspace](../product/agent-workspace-v1-acceptance.md)
rastreia o fechamento deste requisito.

## Gate de escopo integral — Agent Workspace v1

A meta Nexus 1.0 inclui **todas as 48 obrigações** do
[contrato de aceite](../product/agent-workspace-v1-acceptance.md), não uma
seleção arbitrária das features já presentes no código. O Signed Release
bloqueia até existir um artifact
`nexus-v1-workspace-acceptance-<SHA>/evidence.json`, emitido **pelo CI do
próprio SHA**, cobrindo cada feature, duas sessões/clientes nos casos aplicáveis
e as plataformas prometidas.

`node scripts/verify-agent-workspace-v1.mjs` valida a integridade do catálogo
durante o desenvolvimento e **não representa GO**. A chamada `--release`
exige evidence externa, run CI correto e SHA exatamente igual ao checkout.
Não se pode substituir essa evidência por um JSON versionado no repositório.
O produtor E2E do artifact ainda é pendência: **até sua entrega a release fica
NO-GO intencionalmente**.

## Stable v1

Além do CI nativo exato do SHA, a stable exige o workflow de certificação do
mesmo SHA. O gate executável cobre lifecycle assinado (install, upgrade,
self-update, rejeição de tamper e uninstall), provider/Mission reais e soak
cumulativo de 8 horas dividido em segmentos compatíveis com o limite de duração
por step do GitHub Actions.

Somente promova a release estável quando o status autoritativo mudar com
evidência correspondente. Até lá, documentação do source candidato não deve ser
interpretada como anúncio de GA.

## Gate de licenciamento para abertura open source

A meta de licença **Apache-2.0 ainda não está em vigor**. Uma release
comunitária anunciada como open source exige, além dos gates técnicos,
comprovação de titularidade, auditoria jurídica/supply-chain, assets e
histórico, mudança explícita de `LICENSE`, atualização de notices,
READMEs/metadata e autorização formal de distribuição. Consulte
[OPEN_SOURCE_TRANSITION](../legal/OPEN_SOURCE_TRANSITION.md) e
[PUBLICATION_CHECKLIST](../PUBLICATION_CHECKLIST.md). Uma build verde não
substitui a autorização para relicenciar código de terceiros.

## Auditoria de exposição do source já público

Com todos os refs acessíveis localmente, execute `git fetch --all --tags --prune` e `bash scripts/audit-publication.sh`. O script usa Gitleaks, e falha quando existem paths internos a classificar. Um PASS automático não dispensa a revisão humana de histórico, dados pessoais, licenças ou dependências. Nunca envie relatórios brutos de segredos para CI ou issues.

## Evidência visual

Recapturar [screenshots](visual-evidence.md) no SHA congelado antes de usar imagens como prova de funcionamento do Desktop/HUD. Não inserir manualmente um SHA novo em um manifesto antigo.
