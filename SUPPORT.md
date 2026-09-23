# Suporte

IAPro Nexus está em fechamento da v1; suporte de release permanece condicionado ao status autoritativo e à matriz de plataformas. Para dúvidas reproduzíveis, use o issue
tracker do repositório com:

- sistema operacional e arquitetura;
- versão/commit de `nexus version`;
- comando executado;
- resultado esperado e observado;
- logs redigidos de segredos.

Para bugs de segurança, siga [`SECURITY.md`](SECURITY.md) e não use uma issue
pública.

Antes de abrir um issue, consulte o README, o [guia de suporte de
plataformas](docs/operations/platform-support.md) e `nexus doctor`.
Claims de suporte nativo devem ser conferidos na matriz; cross-compilation não
é evidência de runtime nativo.


## Antes de abrir uma issue

Consulte, nesta ordem:

1. [Quick Start](docs/quick-start.md);
2. [FAQ](docs/faq.md);
3. [Troubleshooting](docs/operations/troubleshooting.md);
4. `nexus doctor --json`;
5. [status de release](DEV/validation/current/RELEASE_STATUS.md) quando a dúvida for sobre artefatos/certificação.
