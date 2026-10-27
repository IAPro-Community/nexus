# NEX — Regras de identidade para coding e design agents

Estas instruções complementam o `AGENTS.md` da raiz. **Antes de produzir qualquer arte NEX no app, site, HUD, vídeos ou documentação**, leia:

1. `docs/design/nex/IDENTITY_GOVERNANCE.md`
2. `docs/design/nex/identity-lock.json`
3. `docs/design/NEX_32BIT_VISUAL_CONTRACT.md`
4. `docs/design/NEX_ANIMATION_CONTRACT.md`
5. `docs/design/assets/nex/README.md`

Regras de execução:

- Se `status=CANDIDATE_PENDING_APPROVAL`, **não** promova referências e geração de IA como modelo final, nem altere o personagem canônico. Apresente comparação e registre desvios.
- Se `status=FROZEN`, use **exclusivamente** a versão de identidade e os masters/hashes referenciados. Sem autorização para mudar silhueta/anatomia/contagens por conveniência técnica, estilo ou prompt.
- No site, reinterprete **materiais e apresentação**; nunca o personagem. No HUD, preserve estados ancorados em telemetria real e o mesmo contrato anatômico.
- Em qualquer PR ou arte: informar `identity_lock_version`, origem dos assets, output, escala de uso, tabela de diferenças, evidência lado a lado e testes. Quando não houver master, escrever `UNVERIFIED`, não `APPROVED`.
- Mudança de identidade exige PR separado, before/after, registro e aprovação humana explícita. Solicitação de animação/marketing não autoriza essa mudança.
- Não usar o mesmo arquivo raster para simular 3D polido e pixel-art sem adaptação ótica; preservar topologia e proporções em ambos os masters derivados.
