# Matriz de funcionalidades

Esta matriz descreve presença funcional no **source candidato**. Ela não
substitui certificação de release, evidência de provider autenticado ou execução
nativa em cada sistema operacional.

| Feature | Web | CLI | Desktop | API local | Estado |
| --- | ---: | ---: | ---: | ---: | --- |
| Projects/workspaces | ✅ | ✅ | via Core | ✅ | Disponível |
| Agents persistentes | ✅ | ✅ | via Core | ✅ | Disponível |
| Direct + terminal | ✅ | ✅ | depende de evidência nativa | ✅ | Disponível |
| Providers/profiles/quota | ✅ | ✅ | via Core | ✅ | Disponível com qualidade de evidência |
| Composer | ✅ | operações relacionadas | via Core | ✅ | Disponível |
| WorkPlan / Flow | ✅ | ✅ | via Core | ✅ | Disponível |
| MissionRunner | ✅ | ✅ | via Core | ✅ | Implementado no source candidato |
| Coordination requests | ✅ no Flow Run | indireto via Mission | via Core | ✅ | Implementado no source candidato |
| Mission Teams | ✅ projeção | ✅ | via Core | ✅ | Implementado no source candidato |
| Maestro | ✅ | ✅ | via Core | ✅ | Opcional |
| Update | ✅ | ✅ | depende de pacote/plataforma | ✅ | Disponível com gates de integridade |
| Tunnel | ✅ | ✅ | — | ✅ | Disponível; operação sensível |

## Interpretação

- **Disponível**: contrato e implementação presentes.
- **Implementado no source candidato**: funcionalidade existe no fechamento v1,
  mas a autorização de release estável ainda depende dos gates de certificação.
- **via Core**: Desktop não implementa regra própria; consome o mesmo Core.
- **depende de evidência nativa**: não inferir suporte de cross-compilation.

Veja [suporte por plataforma](../operations/platform-support.md) e o
[status público da release](../release-status.md).
