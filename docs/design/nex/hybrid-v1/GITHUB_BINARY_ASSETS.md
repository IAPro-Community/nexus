# NEX — Assets originais versionados com proveniência

**Estado de fonte:** todos os 13 arquivos runtime do kit híbrido (11 PNGs, HTML e manifesto JSON) estão versionados como blobs Git binários. `build_layers.py` também está versionado. A fonte editável `nex_hybrid_master.ora` é preservada **exatamente** em nove segmentos base64 em `chunks/` porque o conector GitHub limita o tamanho de uma operação binária. Os nove segmentos são parte do repositório, e a reconstrução compara SHA-256 do arquivo completo (5.339.623 bytes).

```bash
node scripts/restore-nex-hybrid-master.mjs
node scripts/restore-nex-hybrid-master.mjs --restore
node scripts/verify-nex-hybrid-assets.mjs --require-binaries
cd web && npm run art:nex:hybrid:qa
```

**Original selecionado:** `web/public/nex/hybrid-v1/nex_source_original.png`, SHA-256 `ef7a4aeb7b24d2ab5f199f913d351414b8fafb0b6208b3ab2b665013adcb9118`. O arquivo foi transportado como blob Git binário e não foi recodificado.

**Atenção:** versionar os assets **não** equivale a aprovação do personagem e não autoriza substituir o mascote da release/HUD nem a página comercial. A fidelidade da animação, o redesenho pixel art Compact, as 12 ações definitivas e as 16 vistas de giro 360° continuam condicionados à avaliação visual e testes nativos. O `RELEASE_READINESS.json` permanece `BLOCKED`.

**Uso no site:** o laboratório de QA pode consumir cópias verificadas desses arquivos. Não inserir o NEX no Hero público até os 11 gates do contrato conjunto passarem.
