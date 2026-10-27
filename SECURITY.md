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

## PRs externos, CI e isolamento de mantenedores

A pipeline `.github/workflows/community-pr.yml` executa propostas externas
em runners efêmeros `ubuntu-24.04`, com `contents: read`, checkout sem
credenciais persistentes, sem ambientes de assinatura e sem uso de secrets.
O workflow `ci.yml` com jobs self-hosted deve receber somente pushes/dispatches
autorizados; **não executar patches de forks em runners persistentes**.

Além dos arquivos do repositório, proprietários devem habilitar **rulesets
e revisão obrigatória** nas branches `main` e `release/v1.0.0`, exigir
os checks apropriados para PRs e limitar quem pode acionar ambientes com
segredos. Uma alteração em YAML **não altera as permissões administrativas
do GitHub**. O canal privado de reporte de vulnerabilidades precisa ser
ativado nas configurações do repositório, se disponível. Nunca inserir
segredos de assinatura em variáveis de PRs externos. Para realizar e
verificar essas mudanças administrativas, consulte
[o playbook de segurança do GitHub](docs/community/ADMIN_SETUP.md).

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
