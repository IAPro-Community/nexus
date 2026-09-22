# Segurança

## Reporte responsável

Não abra uma issue pública para vulnerabilidades não corrigidas. Use um canal
privado de segurança do GitHub, quando habilitado, ou contate os maintainers por
um canal privado.

Inclua, quando for seguro:

- versão ou commit afetado;
- plataforma/ambiente;
- passos mínimos para reproduzir;
- impacto observado;
- sugestão de correção, se houver.

Nunca envie tokens, chaves privadas, cookies, dumps de credenciais, prompts
privados ou dados de usuários. Redija esses valores antes de compartilhar
evidência.

## Modelo operacional

O Nexus executa coding agents e, portanto, trata side effects como uma boundary
de segurança.

Princípios:

- fail closed quando não há prova suficiente;
- menor privilégio;
- worktree/isolamento para escrita autônoma;
- reviewers/investigators read-only quando suportado;
- push/deploy sem permissão implícita;
- dispatch externo desconhecido sem retry cego;
- evidence sanitizada;
- UNKNOWN nunca vira sucesso.

Detalhes: [Guardrails](docs/architecture/guardrails.md).

## API Web

A API é local e autenticada. O bootstrap token é sensível e temporário. Não
exponha a API local como serviço público sem uma arquitetura adicional de
autenticação/autorização/rede.

## Release e supply chain

O source em fechamento da v1 não equivale automaticamente a release estável
certificada.

Assinatura, checksums, notarização e certificação nativa devem ser afirmados
somente quando a revisão/release correspondente tiver evidência publicada.

Status autoritativo:
[RELEASE_STATUS](DEV/validation/current/RELEASE_STATUS.md).

## Escopo de garantia

Testes automatizados de vulnerabilidade, autenticação, autorização, redaction e
isolamento reduzem risco, mas não constituem certificação externa.
