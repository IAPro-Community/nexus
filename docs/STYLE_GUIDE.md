# Guia editorial da documentação

- Fato verificável vem antes de marketing.
- Uma entidade usa um termo canônico; consulte [GLOSSARY.md](GLOSSARY.md).
- Uma página tem uma finalidade principal.
- README é landing page; manuais detalhados vivem em `docs/`.
- Diferencie **implementado no source**, **beta/experimental**, **planejado** e **UNKNOWN**.
- Diferencie status funcional de certificação de release/plataforma.
- Screenshot deve ser real, sanitizado, identificado no manifesto e ter alt text.
- Diagramas conceituais devem viver em `docs/assets/diagrams/`, preferencialmente SVG.
- Diagramas explicam arquitetura; não substituem evidência de runtime.
- Mermaid pode explicar relações simples, mas não deve duplicar um diagrama canônico sem necessidade.
- Use links relativos e valide-os.
- Nunca exponha secrets, cookies, tokens ou paths privados.
- `DEV/` e `.omx/` são contexto interno/histórico, não onboarding público.
- Se um documento histórico conflitar com a documentação canônica atual, preserve o histórico e corrija o índice/precedência em vez de reescrever evidência antiga.
- Não declare `PASS`, suporte nativo ou release publicada antes da evidência do mesmo candidato/revisão.
