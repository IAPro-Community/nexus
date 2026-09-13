# Operação e suporte

## Operação local

| Necessidade | Comando | Observação |
| --- | --- | --- |
| diagnóstico do ambiente | `nexus doctor` | há `--json` e `--bundle` para evidência sanitizada |
| estado dos providers | `nexus providers status --json` | não publique emails, tokens ou caminhos privados |
| estado dos runtimes | `nexus ps --json` | use antes de recuperar ou encerrar uma sessão |
| estado de perfil | `nexus status [provider:profile]` | quota pode ser `UNKNOWN` ou `CACHED` |
| auditoria de isolamento | `nexus security --json` | revise achados antes de ações de correção |

## Limites de operação

Nexus é uma workstation local. A API da Web é autenticada e destinada às
superfícies do próprio produto; não a exponha como um serviço público. Para
acesso remoto, trate [tunnel](../cli/README.md#tunnel) como operação consciente
de risco e confirme a política de segurança antes de usá-lo.

Consulte [troubleshooting](troubleshooting.md), [segurança](../security.md) e
[suporte de plataformas](platform-support.md).
