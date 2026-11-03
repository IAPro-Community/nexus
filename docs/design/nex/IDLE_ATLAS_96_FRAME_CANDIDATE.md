# NEX — atlas 2D de 96 quadros (candidato experimental)

O protótipo de 12 idles recebeu um atlas de **1024 × 1536 px**, contendo **12 ações × 8 quadros × 128 × 128 px**, com manifesto por ação/quadros/125ms. A versão do ZIP `NEX_12_Idle_Atlas_Candidate.zip` possui SHA-256 `3b4c53dd8e9f1b19ad1ac6e59eb385df84f22371bc7a27563a0ad4d48be6d82c`; atlas `nex-idle-atlas-128.png` SHA-256 `8256a302fff77996e3563ef42d1e62685b59084c8a17446bb2acb2219e85dfda`.

**QA realizado:** Chromium abriu uma prévia HTML independente, com atlas decodificado corretamente em 1024px, 12 ações selecionáveis, troca para `idle_blink` com evolução real de frame 4/8 e parada imediata por checkbox. Nenhum erro JavaScript na execução observada. A storyboard de 12 linhas permite examinar todos os quadros lado a lado.

**Limitação artística fundamental:** estes 96 quadros são um **estudo de movimento em camadas extraídas do PNG original**, reconstruído por deslocamento, escala e rotação pequena, e **não** 96 sprites desenhados manualmente. Algumas ações são quase imperceptíveis a 128px; especialmente aletas, inclinação e curiosidade exigem rig de cabeça, novas formas de olhos e preenchimento manual das áreas ocultas. Este atlas **não está pronto para HUD**.

**Atualização 29/10/2026:** o PNG do atlas, o storyboard e o manifesto de 96 quadros estão versionados na [release](https://github.com/IAPro-Community/nexus/tree/release/v1.0.0/docs/design/nex/experimental-atlas) pela PR #147. O pacote continua **CANDIDATE_NOT_APPROVED**, e `RELEASE_READINESS.json` permanece `BLOCKED`: versionar um atlas de estudo não o torna master retrô aprovado.

**Para produção:** redesenhar optical-specific os 64/128px a partir do master aprovado, refazer os 12 ciclos como quadros de sprite consistentes e validar em Windows/macOS/Linux. Integrar o `NexIdleDirector` apenas quando o atlas for aprovado. O giro 360° real **não pode ser construído** deste atlas frontal: requer 16 vistas próprias (ângulos 0°–337,5°).
