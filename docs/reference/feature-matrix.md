# Matriz de funcionalidades

Os estados descrevem a presença da capacidade no produto. Eles não substituem a
evidência de uma conta autenticada ou de um sistema operacional específico.

| Feature | Web | CLI | Desktop | API local | Status |
| --- | ---: | ---: | ---: | ---: | --- |
| Projetos e workspace | ✅ | ✅ | quando a shell nativa estiver disponível | ✅ | Disponível |
| Agents persistentes | ✅ | ✅ | quando a shell nativa estiver disponível | ✅ | Disponível |
| Sessão Direct e terminal | ✅ | ✅ | evidência por plataforma | ✅ | Disponível |
| Providers, perfis e quota | ✅ | ✅ | quando a shell nativa estiver disponível | ✅ | Disponível com estados de evidência |
| Composer | ✅ | por plan/compile | evidência por plataforma | ✅ | Disponível |
| Flow / WorkPlan | ✅ | ✅ | evidência por plataforma | ✅ | Disponível |
| Mission / execução verificada | ✅ | `nexus run` / plan | evidência por plataforma | ✅ | Experimental |
| Delegação automática do Lead | sessão interativa | `nexus <provider> --supervised` | não certificada separadamente | ✅ | Experimental |
| Maestro opcional | ✅ | ✅ | evidência por plataforma | ✅ | Opcional / Experimental |
| Atualização | ✅ | ✅ | depende do pacote | ✅ | Disponível com requisitos de assinatura/pacote |
| Tunnel remoto | ✅ | ✅ | — | ✅ | Disponível; requer revisão de segurança |

Veja [suporte por plataforma](../operations/platform-support.md) para limites
de Linux, macOS e Windows, e [evidência de documentação](../internal/DOCUMENTATION_EVIDENCE.md)
para a origem de cada afirmação.
