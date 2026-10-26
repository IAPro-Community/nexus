# NEX Identity Lock — modelo final, derivação e governança

> **Estado atual: CANDIDATE_PENDING_APPROVAL.** As três pranchas fornecidas pelo usuário em 26/10/2026 são referências de exploração, **não** a nova arte mestre congelada. O contrato v1 existente e o sprite Compact continuam válidos. Não substituir a identidade oficial por experimentos.

## Intenção

Eliminar deriva visual: **uma identidade, múltiplas representações**. Uma peça para site ou vídeo não autoriza outro agente a redesenhar o NEX. O estilo comercial polido e a interpretação 32-bit devem preservar silhueta, proporções-chave, visor, duas aletas, dois acoplamentos laterais, um núcleo, quatro módulos orbitais e anel inferior. As imagens de referência são editoriais: nenhum dos textos impressos dentro delas altera os contratos técnicos do produto.

## Fonte de autoridade e ordem

1. Após homologação, `identity-lock.json` (status `FROZEN`) + masters editáveis e binários versionados, com SHA-256; essa combinação define **quem é o NEX**.
2. O contrato existente `../NEX_32BIT_VISUAL_CONTRACT.md` define anatomia permitida e os limites de runtime; não reclassificar situações falsas como estados reais.
3. `../../..//web/src/features/hud/NexPixelSprite.tsx` (caminho do root: `web/src/features/hud/NexPixelSprite.tsx`) representa a implementação Compact da release, não um master Hero.
4. Pranchas de referência (incluindo JPEGs do usuário), drafts, mockups e saídas de IA **não** são masters; não promover sem aprovação.

## Fluxo de aprovação e congelamento

1. Conservar as três referências originais sem recompressão/recorte. O manifesto atual guarda nome, resolução e hash; os bytes ainda não estão no repo.
2. Produzir candidatos com vistas frontal, 3/4, lateral e traseira; expressões neutra, trabalho, atenção, crítico e offline; preview sem cenário e silhueta em 32/64/92/256 px. Apresentar lado a lado com as referências.
3. A escolha final exige **aprovação expressa do usuário**, registrando data, identidade do aprovador e commit. Aprovar uma imagem de marketing não comprova homologação nativa.
4. Versionar `masters/` com fonte editável e export de referência sem perdas; preencher `master.file`, `master.sha256`, `master.editable_source_file`, `master.editable_source_sha256` e `locked_assets` no `identity-lock.json`.
5. Atualizar `status: FROZEN`, `proposed_lock_version: NEX-ID-v1` e `approval_record`. Executar `node scripts/verify-nex-identity-lock.mjs`; manter registro de arte aprovada e uma revisão humana de paridade.
6. Congelar alterações em master, invariantes ou hashes: **somente PR de mudança de identidade**, com registro `docs/design/nex/change-requests/<versao>.md`, antes/depois, justificativa, migração site/HUD e aprovação explícita. Exigir revisão de CODEOWNERS e branch protection no GitHub para enforcement organizacional.

## O que agentes podem mudar sem redesenhar

- **Pode:** expressões dentro do mesmo visor, estados operacionais previstos, iluminação, material, pixelização, escala, cenário, pequenas poses de suporte, motion em pivôs autorizados, cores contextuais limitadas. Registrar sempre o asset base e o lock version.
- **Não pode:** trocar personagem, visor, aletas, orelhas, núcleo, quantidade/posição dos quatro módulos, proporção cabeça/corpo, criar braços e pernas humanos, adicionar boca, reproduzir outro robô a partir de texto, modificar o logo institucional.
- **Quando faltar master:** trabalhar só em wireframe ou arte rotulada **EXPLORAÇÃO NÃO OFICIAL**, nunca publicar como novo NEX definitivo.
- **Agentes como papeis:** Planner/Builder/Reviewer/Guardian (se ilustrados) podem ter sinalizações e cores, mas não uma espécie distinta de NEX. Não afirmar que NEX coordena esses agentes no runtime.

## Interoperabilidade site → HUD

Não exigir arquivo de imagem idêntico entre 3D polido e pixel art (tecnicamente impossível sem perder acabamento). Exigir **mesma topologia, silhueta, proporções, marcos de anatomia, expressões semanticamente compatíveis e asset derivado de master identificado**. Compartilhar esquema de states e pivôs. Futuro 3D no HUD depende de protótipo e desempenho próprios; não prometer suporte antes de integrar.

## Mecanismos de qualidade

- `node scripts/verify-nex-identity-lock.mjs`: valida schema/status, hashes e presença de master/assets quando `FROZEN`. Em modo candidato sinaliza pendência, nunca certifica paridade.
- `node scripts/verify-nex-visual.mjs`: valida restrições estruturais existentes; não reconhece semelhança perceptiva.
- Comparação humana obrigatória em fundo neutro, 32/64/92/256 px, claro/escuro, sem glow/partículas, e animação desligada.
- O branch protection e CODEOWNERS são necessários para impedir merges arbitrários: documentação/CI isolados não garantem bloqueio administrativo.

## Inventário das 3 referências fornecidas

Os três JPEGs de 1448 × 1086 enviados pelo usuário estão identificados com SHA-256 em `identity-lock.json`. **Não foram importados ao GitHub nesta PR.** Até a ingestão do material aprovado, eles são referências de procedência e não masters oficiais. Não confundir com o PNG previamente documentado em `NEX_VISUAL_MANIFEST.json` (outra origem e outro hash).
