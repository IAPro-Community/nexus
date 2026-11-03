# NEX — Pranchas separadas de referência para pixel art retrô

**Direção selecionada:** robô branco/azul da prancha enviada pelo usuário em 27/10/2026, interpretado como personagem 32-bit-inspired de **alta densidade de pixels, com várias ações idle**. Este diretório documenta um novo conjunto de dez imagens — uma prancha integral e nove recortes do mesmo desenho — e especifica a incorporação binária, garantindo consistência visual entre anatomia, expressões e idles.

**Atenção:** as imagens deste pacote foram **geradas nesta conversa**; não são a referência original enviada pelo usuário e **não são quadros válidos de animação**. Podem conter microvariações, rótulos imperfeitos e frames incompatíveis. O original escolhido pelo usuário segue registrado em `../identity-lock.json.selection_record` e tem precedência em caso de conflito.

## Coleção atual para comparar poses (29/10/2026)

A [nova coleção de seis vistas em 256 px](turnaround-v1/README.md) substitui **apenas a indicação de comparação de perspectivas anteriores**. Inclui arte frontal, dois desvios leves, dois ângulos de 3/4 e costas, com [manifesto SHA-256](turnaround-v1/manifest.json), [prancha](turnaround-v1/contact-sheet.webp) e página de revisão. A arte é 256px WebP lossless, derivada de seis imagens de 1254px geradas independentemente.

**Não** substituir o master ou a arte de HUD por esse material ainda: ele tem fundo opaco, ângulos intermediários não certificados e nenhum giro 360° pronto. Os conjuntos anteriores continuam preservados para auditoria e comparação histórica.

## Inventário do pacote

| Imagem | Finalidade |
| --- | --- |
| `00_nex_guia_conceitual_completo.png` | Prancha editorial completa |
| `01_nex_personagem_principal.png` | Proporções e acabamento do NEX |
| `02_nex_vistas_frente_34_perfil_costas.png` | Turnaround em vistas auxiliares |
| `03_nex_expressoes_visor.png` | Expressões projetadas no visor |
| `04_nex_detalhes_anatomia.png` | Anatomia e material metálico |
| `05_nex_idle_12_acoes.png` | Doze gestos idle conceituais |
| `06_nex_estados_operacionais.png` | Cinco estados operacionais |
| `07_nex_acoes_interacao.png` | Ações e gestos ilustrativos |
| `08_nex_contextos_de_uso.png` | Cenários HUD, Docs e site |
| `09_nex_paleta.png` | Paleta editorial |

O `manifest.json` guarda dimensões, bytes e SHA-256 **calculados sobre os arquivos reais**. Todos os derivados são recortes sem reinterpretação da mesma prancha; assim, não há nove redesenhos autônomos que provoquem deriva de personagem.

## Avaliação de paridade mais recente (28/10/2026)

Uma nova [prancha de auditoria e PNG Hero isolado](2026-10-28/README.md) registra a diferença identificada entre o SVG de site e a imagem de referência. Os arquivos estão disponíveis no chat em um ZIP e os metadados e SHA-256 estão no [manifesto](2026-10-28/manifest.json); não estão versionados como PNG no GitHub enquanto o manifesto indicar `PENDING_BINARY_UPLOAD`.

## Estado dos binários no GitHub

**`PENDING_BINARY_INGEST`** — a criação de uma lista de nomes e hashes em um commit GitHub **não significa que as imagens PNG tenham sido commitadas**. Os binários são entregues neste chat como arquivos e como pacote ZIP; devem ser copiados para este diretório no repositório e verificados com:

```bash
node scripts/import-nex-reference-pack.mjs --source /caminho/extraido/nex_retro_references_2026_10_27
node scripts/verify-nex-reference-pack.mjs
```

O primeiro comando valida todos os PNGs contra o manifesto antes de copiá-los, e então muda o estado dos arquivos para `VERSIONED`; o segundo é o gate permanente de integridade. **Nunca substituir PNGs por miniaturas ou URLs efêmeras.**

## Hierarquia e fidelidade

1. [Direção original selecionada pelo usuário](../identity-lock.json) — autoridade da identidade.
2. [Direção Retro Game e 12 idles](../RETRO_GAME_IDLE_DIRECTION.md) — aparência, coreografia e requisitos do atlas.
3. [Contrato de anatomia visual](../../NEX_32BIT_VISUAL_CONTRACT.md) — invariantes.
4. **Imagens deste pacote** — estudos complementares para arte/expressões, não master bloqueado.
5. Código `web/src/features/hud/NexPixelSprite.tsx` — fallback vetorial funcional, não prova de pixel art HD final.

## Requisitos para a próxima arte

- Desenhar **um master editável em camadas** (não extração de frames destas pranchas) com casco, visor, duas aletas, corpo, núcleo, quatro módulos e órbita.
- Construir sprites Compact (64px), Standard (128px) e Hero (256px+) com revisão óptica, pivôs e atlas JSON.
- Adicionar os 12 idles ao catálogo em etapas; eventos do Core interrompem gestos não operacionais imediatamente.
- Conferir consistência quadro a quadro, inclusive em movimento reduzido e em `offline`.
- A arte definitiva só vira `FROZEN` após arquivo editável, hashes, comparação com o original escolhido, testes e aprovação explícita.

A aprovação do personagem não equivale à homologação do binário da release v1.
