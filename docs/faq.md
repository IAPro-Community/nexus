# FAQ

## Preciso usar Composer, Flow ou Mission para conversar com um Agent?

Não. O caminho Direct continua sendo:

```text
Project → AI Session → Provider → Terminal
```

Use planejamento/orquestração apenas quando trouxer clareza ou controle.

## Qual é a diferença entre Agent e provider?

Agent é uma identidade persistente com papel/histórico. Provider é o recurso que
executa uma geração concreta. Trocar provider não precisa trocar a identidade
do Agent.

## Flow é outro formato de plano?

Não. WorkPlan é a fonte de verdade e Flow é sua projeção visual/editável.

## O que acontece quando pressiono GO?

O Nexus autoriza/congela uma revisão do plano e inicia uma Mission sobre essa
revisão. Mudanças materiais posteriores não devem ser aplicadas
silenciosamente.

## Quando uma Mission está realmente concluída?

Quando packages obrigatórias estão verificadas, não existe intervenção pendente
e o Definition of Done global passa. O estado final é
`COMPLETED_VERIFIED`.

## Agents conversam livremente entre si?

Não. Coordenação é tipada e passa pelo MissionRunner:

- `NEED_INFO`;
- `ARTIFACT_REQUEST`;
- `REVIEW_REQUEST`;
- `DELEGATE`.

Veja [Coordenação entre Agents](product/agent-coordination.md).

## O Nexus repete automaticamente uma package quando o Agent pede informação?

Somente quando consegue provar que o replay é seguro. Mudança do workspace ou
possibilidade de side effect externo faz a Mission bloquear para intervenção
humana.

## O Nexus é o provider de IA?

Não. Nexus opera CLIs/providers e seus profiles. A autenticação permanece
pertencendo ao provider oficial.

## Maestro é obrigatório?

Não. Maestro adiciona método, skills, risco e gates. Direct, Agents, Composer,
Flow e Mission pertencem ao Nexus.

## Posso assumir que quota está disponível quando ela não aparece?

Não. `UNKNOWN`, `CACHED`, `ESTIMATED` e `LIVE` têm significados
diferentes. Ausência de evidência não equivale a capacidade disponível.

## A API `/api/v1` é pública?

Não. É uma API local usada pelas superfícies do produto. Para automação externa,
prefira CLI e `--json` quando disponíveis.

## O Desktop é suportado em qualquer sistema?

Não automaticamente. Build e runtime nativos precisam de evidência da
plataforma. Consulte [suporte de plataformas](operations/platform-support.md).

## O source v1 significa que a release v1 está publicada?

Não. Implementação funcional e certificação de release são gates distintos.
Consulte [status da release](release-status.md).

## O Nexus já é open source? Posso criar um fork e distribuir?

**Ainda não.** O código é público, mas a licença vigente PolyForm Strict
não permite genericamente modificar ou redistribuir o produto. A licença
**Apache-2.0 é o objetivo**, ainda dependente de auditoria de direitos
autorais, componentes de terceiros e segurança do histórico. Até o
relicenciamento autorizado, sugerimos participar por issues e feedback de
produto/documentação. Leia [plano de abertura](legal/OPEN_SOURCE_TRANSITION.md).

## Quando aberto, serei obrigado a enviar meu fork de volta?

Sob a licença-alvo permissiva Apache-2.0, não. Queremos atrair melhorias
pela qualidade do upstream, documentação, processo de revisão e comunidade,
não por imposição de retorno dos forks. O fork deverá respeitar notices,
termos de software e identidade visual, conforme as licenças aplicáveis.

## O Composer, Flow e múltiplos agentes serão pagos?

A **Community planejada** manterá essas capacidades funcionais sem uma
assinatura obrigatória do Nexus. Provedores externos de IA podem cobrar por
suas próprias APIs, modelos, contas ou limites. Patrocínios, serviços
gerenciados e suporte empresarial serão opcionais. Não há serviço
comercial ou programa de doações ativo implicitamente anunciado aqui.

## Como reporto um erro sem expor dados?

Use `nexus doctor --bundle` quando apropriado, remova tokens, emails, paths e
nomes privados, e siga [Troubleshooting](operations/troubleshooting.md).
