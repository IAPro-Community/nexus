# NEX Hero híbrido — implementação de fidelidade v1.1 (P0)

## Atualização — binários agora realmente versionados no Git (29/10/2026)

A fonte PNG original (SHA-256 `ef7a4aeb7b24d2ab5f199f913d351414b8fafb0b6208b3ab2b665013adcb9118`), as 8 camadas individuais, os PNGs de base/reconstrução, o HTML e o gerador `build_layers.py` **foram adicionados como blobs binários Git** nesta branch. O atlas experimental de 96 quadros também foi incluído em `docs/design/nex/experimental-atlas/`, ainda sem aprovação artística.

A fonte editável OpenRaster (5.339.623 bytes) é **reconstituível byte a byte** a partir de seis partes base64 versionadas em `docs/design/nex/hybrid-v1/ora-parts/`. Para recuperá-la, executar:

```bash
node scripts/restore-nex-openraster.mjs
node scripts/verify-nex-hybrid-assets.mjs --require-binaries
```

O verificador valida as seis partes diretamente em memória, mesmo antes da restauração. O SHA-256 reconstruído deve ser `68a9b3b90722721bacd4067bac3e9d6560acf144898ae60c3d7968692c652f32`.

**Atenção:** versionamento correto dos bytes **não é homologação da identidade**, não significa que o master pixel art Compact/Standard esteja pronto e não libera o NEX no site comercial. O gate conjunto `RELEASE_READINESS.json` permanece `BLOCKED`. O estágio local `/nex-lab` pode ser usado somente para QA.



**Objetivo:** manter a aparência real do PNG de NEX (`mascote_robô_nexus_em_neon_azul.png`, 1254×1254, transparência) e obter movimentos leves **sem tentar redesenhar o personagem em SVG**.

## Entregas realizadas no ambiente de produção de arte

- `nex_source_original.png`: PNG original **idêntico byte a byte ao arquivo aprovado** (não apenas igual em pixels). SHA-256 `ef7a4aeb7b24d2ab5f199f913d351414b8fafb0b6208b3ab2b665013adcb9118`; 1.679.808 bytes. O pacote final foi revisado para evitar recodificação PNG acidental.
- `body_base.png`: base com visor escurecido sob os olhos e módulos separados.
- `eye_left.png`, `eye_right.png`: olhos extraídos do PNG original para piscada e olhar.
- `module_1.png` até `module_4.png`: peças móveis recortadas sem redesenho, pivôs no `manifest.json`.
- `fin_left.png`, `fin_right.png`: reflexos localizados nas aletas, para microajustes de brilho (não geometria 3D real).
- `nex_hybrid_master.ora`: fonte editável OpenRaster em camadas para GIMP/Krita.
- `index.html`: QA visual independente com cinco estados e as seis microações P0.
- `reconstructed_idle.png`: imagem composta a partir das camadas.

**QA da reconstrução v1.1 (fonte original preservada byte a byte):** testes reais no Chromium encontraram **áreas retangulares atrás dos olhos** durante a piscada, apesar da boa aparência em repouso. A base do visor foi refeita com preenchimento navy suave e transparência restaurada, sem modificar o PNG original. A diferença média contra a referência passa a **0,787/255 nos canais RGB** e **0,168/255 no alpha**; aproximadamente **12,31% dos pixels** da recomposição têm alguma diferença, em geral localizada no visor. A escolha prioriza uma piscada visualmente correta em vez de paridade RGB estática absoluta. **Aprovação perceptual humana permanece pendente.**

## Animações P0 — escopo real

| Ação | Implementação de protótipo | Limite |
| --- | --- | --- |
| `idle_hover` | deslocamento de até 5px do conjunto | não altera a janela do HUD |
| `idle_blink` | escala vertical dos próprios olhos extraídos | visor sob os olhos tem preenchimento reconstruído |
| `idle_glance` | deslocamento pequeno dos olhos | não produz pupilas 3D |
| `idle_orbit_breathe` | quatro módulos separados com deslocamentos locais | sem troca de oclusão real |
| `idle_head_tilt` | rotação de ±1,8° do personagem completo | **a cabeça ainda não tem rig totalmente separado** |
| `idle_fin_twitch` | reflexos das aletas oscilam | **não é dobramento geométrico** |

`idle_head_tilt` e `idle_fin_twitch` são **protótipos** de linguagem de movimento. A evolução definitiva requer recorte com preenchimento da cabeça/fin próprio e validação visual.

## Importação automatizada do ZIP original

Existe agora `scripts/import-nex-hybrid-package.py` (Python padrão, sem bibliotecas externas), que **verifica todos os 14 assets e o gerador antes de escrever qualquer arquivo**, sem recodificar o PNG e sem criar commits automaticamente:

```bash
python scripts/import-nex-hybrid-package.py /caminho/nex_hybrid_github_assets_v1_1.zip
node scripts/verify-nex-hybrid-assets.mjs --require-binaries
git add web/public/nex/hybrid-v1 docs/design/nex/hybrid-v1
git commit -m "assets(nex): import source-faithful Hero v1.1 layers"
git push
```

O pacote original de 15 arquivos foi importado e verificado; **não é necessário obter o ZIP da conversa novamente**. A PR #147 adicionou os PNGs e o gerador como arquivos reais ao GitHub. O OpenRaster editável é recuperado dos seis fragmentos versionados executando `node scripts/restore-nex-openraster.mjs`, com SHA-256 verificado. O manifesto de origem indica arquivos versionados ou reconstruíveis, sem atribuir aprovação artística.

## Entrega ao GitHub / integração

**Fonte canônica de trabalho:** `web/public/nex/hybrid-v1/` e `docs/design/nex/hybrid-v1/` na `release/v1.0.0`. Para recuperar a fonte editável OpenRaster, usar `node scripts/restore-nex-openraster.mjs`. Para validar bytes do checkout, `node scripts/verify-nex-hybrid-assets.mjs --require-binaries`. Não promover a arte ao site/HUD público até aprovação dos gates de movimento, pixel art e 360°.

```bash
# Em um checkout da branch release/v1.0.0 ou uma branch de trabalho:
unzip -o /caminho/nex_hybrid_github_assets_v1_1.zip -d .
node scripts/verify-nex-hybrid-assets.mjs --require-binaries
node scripts/verify-nex-hybrid-assets.mjs --promote
git add web/public/nex/hybrid-v1 docs/design/nex/hybrid-v1/asset-manifest.json docs/design/nex/hybrid-v1/nex_hybrid_master.ora docs/design/nex/hybrid-v1/build_layers.py
git commit -m "assets(nex): import verified layered Hero PNGs and original"
git push
```

Antes da ingestão, `asset-manifest.json.status=BINARY_INGEST_PENDING_VISUAL_APPROVAL`; depois do `--promote`, o status passa a `ASSETS_VERSIONED_PENDING_VISUAL_APPROVAL`. **A promoção dos arquivos não promove o personagem como master artístico final.** A revisão v1.1 inclui um gerador portável em `docs/design/nex/hybrid-v1/build_layers.py`; os PNGs exportados foram reproduzidos e comparados byte a byte, mas o arquivo `.ora` pode diferir em metadados ZIP quando regenerado.

Após importar os binários e instalar as dependências do web, rode `cd web && npm run art:nex:hybrid:qa`. O comando executa Chromium com o **HTML e PNGs originais versionados**, gera `web/dist/nex-hybrid-qa/` (idle, olhos fechados, offline e resultado JSON) e verifica as cinco projeções, seis ações P0, preempção, modo offline e movimento reduzido. Esse QA é web, não homologa o HUD.

Abra `web/public/nex/hybrid-v1/index.html` por servidor HTTP (ou diretamente no navegador, conforme política de arquivos locais). O protótipo é completamente local, sem rede, sem dependências React/Three.js e sem LLM.

## Critérios de aprovação do Hero para site

1. PNG original e reconstrução lado a lado: silhueta, metal, visor, núcleo e quatro módulos reconhecidamente idênticos em repouso.
2. Verificar as seis ações no Chrome, Firefox, Safari e celular, sem buracos na máscara ocular, halos duplicados ou mudança do número de módulos.
3. `motion=off`, `prefers-reduced-motion`, `offline` e `visibilitychange` suspendem timers/efeitos.
4. Medir transferência inicial, INP/FPS e impacto no LCP; carregar a versão Hero **apenas** na seção do site que realmente a utiliza. Não importar ORA ou debug PNG em produção.
5. Autorizar o uso após revisão explícita de fidelidade; não substituir o HUD Compact antes de atlas retrô 64/128px próprio.
6. Giro de 360° real continua na [issue #129](https://github.com/IAPro-Community/nexus/issues/129); não usar `rotateY` como falso giro.

## Fronteiras entre produtos

- **Nexus (runtime/HUD):** manter o NEX Compact atual enquanto a issue #119 (master pixel art) e #120 (idles) não forem homologadas no HUD nativo.
- **Nexus-Site:** existe um novo estudo Hero híbrido; pode usar o mesmo conjunto de imagens após importação com hash. Não instalar o pacote de produção até os binários estarem realmente presentes e a qualidade ser aprovada.
- **Maestro/Core:** nenhuma mudança; estados `working/warning/critical/offline` seguem baseados em telemetria verdadeira.

## Comparação visual com o SVG anterior

O SVG `web/public/nex/nex-site-hero.svg` permanece candidato; a versão híbrida mantém pixels reais do PNG e deve ser preferida quando a fidelidade à referência for mais importante do que edição vetorial. O SVG pode continuar sendo útil para partículas e órbitas, mas não deve tentar substituir o capacete e rosto reais por caminhos simplificados.
