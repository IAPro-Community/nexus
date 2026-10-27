# Nexus — migração responsável para open source

**Decisão de produto:** o Nexus Community será aberto, utilizável localmente e sem limites artificiais de agentes, Composer, Flow ou Missions. A monetização pretendida é voluntária (patrocínios/doações) e, futuramente, serviços gerenciados, suporte e contratos corporativos. **Licença-alvo proposta: Apache-2.0.**

**Estado jurídico atual (27/10/2026):** `LICENSE` contém **PolyForm Strict 1.0.0**, que não autoriza em geral modificação nem redistribuição e não é open source OSI. **Esta página não concede licença, nem substitui LICENSE.** A transição só entra em vigor por commit de licenciamento aprovado depois dos gates abaixo.

## Direitos que pretendemos conceder, quando aprovados

Apache-2.0 permitirá usar, estudar, executar, modificar, distribuir, vender e criar forks, inclusive comercialmente, mantendo avisos aplicáveis. Sua cláusula de patentes e regras de aviso também se aplicam. A licença não obriga forks a enviar melhorias upstream, nem licencia automaticamente nomes, marcas, logos, identidade visual ou dependências de terceiros.

Não tentar proibir comércio, forks ou hospedagem de terceiros por meio de uma "licença open source": isso violaria a Open Source Definition. Proteger nome e marca separadamente; competir por qualidade, atualizações, segurança, suporte e relação com a comunidade.

Referências:
- https://opensource.org/osd
- https://www.apache.org/licenses/LICENSE-2.0
- https://apache.org/legal/apply-license.html

## Gate P0 — autorização de relicenciamento (somente decisão do titular)

- [ ] Registrar quem detém os direitos patrimoniais do código, documentação, interfaces, assets, logos e nomes. Não presumir que organização GitHub ou autor de commit sejam o titular.
- [ ] Auditar todas as contribuições e bases importadas (incluindo commits anteriores à PolyForm Strict e eventuais trechos gerados ou copiados de terceiros). Mapear licenças anteriores, contratos de trabalho e autorizações.
- [ ] Obter autorizações de cada titular relevante para relicenciar seu conteúdo sob Apache-2.0, ou remover/substituir os trechos cujo licenciamento não for comprovado.
- [ ] Inventariar recursos visuais, imagens geradas, fontes, ícones, dados e binários vendorizados; separar material de terceiros com seus próprios direitos. Não assumir que tudo no repositório entra na licença raiz.
- [ ] Classificar privadamente alertas de segredos do histórico descritos em [PUBLICATION_CHECKLIST](../PUBLICATION_CHECKLIST.md); rotacionar credenciais confirmadas e tratar dados pessoais antes de promover novos meios de distribuição.
- [ ] Gerar SBOM do checkout e dos artefatos distribuídos e revisar licenças transitivas, notices, copyleft e obrigações de distribuição conforme [THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES.md).
- [ ] Obter revisão jurídica do texto final e da competência do signatário; registrar decisão, data, escopo do código e exceções.
- [ ] Só então substituir **inteiramente** `LICENSE` pelo texto oficial Apache-2.0 **sem alterações**, acrescentar `NOTICE` apropriado e exceções identificadas em `THIRD_PARTY_NOTICES`; atualizar docs/testes/metadados/release com o mesmo commit/PR.
- [ ] Reexecutar `go test ./internal/release/...`, `go test ./...`, `bun run quality:full`, auditoria de links/documentação, varreduras privadas de segredos e a matriz de CI exact-SHA.
- [ ] Publicar comunicado de mudança de licença com SHA efetivo. Não declarar relicenciamento retroativo de artefatos distribuídos com termos próprios sem exame específico.

**Gate de decisão:** um mantenedor autorizado registra por escrito a conclusão da matriz de titularidade antes da mudança de LICENSE. Ausência de evidência = NÃO RELICENCIAR, mesmo que o código esteja acessível publicamente.

## Modelo Community e sustentabilidade

- A Community é o produto funcional, sem necessidade de conta, cobrança ou servidor comercial Nexus; CLI/LLM de terceiros podem ter custos ou contratos próprios.
- Composer, Flow, múltiplos agentes, histórico, autorização de ações sensíveis, isolamento e recuperação continuam na Community. Segurança básica não é paywall.
- Doações e patrocínios são voluntários e não compram aprovação de PRs, prioridade de falhas de segurança ou poder de decisão técnico.
- Serviços futuros podem cobrar hospedagem, administração, suporte com SLA, instalação assistida, treinamento e operação corporativa, sem tornar a Community dependente deles.
- Toda mudança de precificação ou de escopo Community requer proposta pública, prazo de comentários e atualização de [GOVERNANCE](../../GOVERNANCE.md).

## Plano de execução público

1. **PREPARAÇÃO — agora:** documentação, processos de issues, governance, playbook de segurança, modelo de decisão e colaboração sem falsas permissões. Pull requests de código de terceiros continuam sujeitos à licença vigente.
2. **CLEARANCE — bloqueado:** titularidade, segredos, licenças de terceiros, assets e revisão jurídica. Material sigiloso da auditoria não deve entrar em issues ou logs públicos.
3. **OPEN SOURCE — após clearance:** commit com Apache-2.0, documentação e testes de licença consistentes, código de conduta, DCO/fluxo de PR, exemplos de upstream, announcement e release associada a SHA.
4. **COMUNIDADE:** issue triage, good first issue, roadmaps, docs e revisão pública; abrir patrocínios quando uma conta real estiver habilitada.
5. **SUSTENTABILIDADE:** medir adoção e demanda por serviço antes de decidir implantação gerenciada e contratos empresariais.

Leia também [Licenciamento atual](LICENSING.md), [diretrizes de fork/upstream](../community/FORKS_AND_UPSTREAM.md), [identidade de marca](TRADEMARK_POLICY.md) e [contribuições](../../CONTRIBUTING.md).
