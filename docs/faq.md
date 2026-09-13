# FAQ

## Preciso usar Composer, Flow ou Mission para conversar com um agent?

Não. O caminho Direct existe para projeto → sessão → provider → terminal. Use
as camadas de planejamento quando elas trouxerem clareza ou controle adicional.

## O Nexus é o provider de IA?

Não. Nexus opera CLIs/providers e seus perfis locais. A autenticação continua
sendo do provider oficial.

## Posso assumir que a quota está disponível quando ela não aparece?

Não. `UNKNOWN`, `CACHED`, `ESTIMATED` e `LIVE` têm significados diferentes.
Consulte [providers e quota](product/providers-and-usage.md).

## A API `/api/v1` é pública?

Não. É uma API local interna usada pelas superfícies do produto. Para scripts,
prefira os comandos CLI que oferecem `--json`.

## O Desktop é suportado em qualquer sistema?

Não automaticamente. A disponibilidade depende do build nativo e de evidência
da plataforma. Consulte [suporte de plataformas](operations/platform-support.md).

## Como reporto um erro sem expor dados?

Rode `nexus doctor --bundle` se apropriado, remova tokens, emails, paths e nomes
de repositórios privados, e siga [troubleshooting](operations/troubleshooting.md).
