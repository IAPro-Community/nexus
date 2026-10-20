# Nexus 1.0.0 — primeira execução e configuração de providers

Escopo: branch `release/v1.0.0`. O HUD não é pré-requisito para cadastro de
providers, login, recursos ou execução. Composer/Flow permanecem ocultos por
padrão na release pública.

## Gates locais obrigatórios

No Windows (PowerShell), em um checkout limpo da release:

```powershell
git checkout release/v1.0.0
git pull --ff-only
cd web
npm run format:check
npm run lint
npm run typecheck
npm run test
npm run build
```

Se houver falhas, registre comando, versão do Node e log. Não substitua um
resultado não executado por PASS.

## Cenários de homologação

| Cenário | Esperado |
| --- | --- |
| Primeira instalação sem CLIs | Projeto pode ser criado; recomenda instruções oficiais sem inventar login |
| CLI instalada sem perfil | Recomenda registro explícito do perfil |
| Perfil pendente | Concluir login da conta existente; não registrar duplicata |
| Perfil expirado | Mesmo fluxo de recuperação, sem nova conta |
| Registro novo | Atualiza lista diretamente do backend; abre login da conta correta |
| Login não concluído | Verificar não informa sucesso e mantém assistente aberto |
| Login de outro perfil | Não confirma o perfil solicitado |
| Login concluído | Backend confirma `authenticated`; cota é tratada separadamente |
| Conta e CLI já prontas | Aviso da Visão geral some; sessão direta permanece acessível |
| HUD desativado | Visão geral e Uso/Contas ainda detectam contas e permitem abrir Project Shell |
| OpenCode local | Não exigir login em nuvem de forma fictícia |
| Windows e Linux | URLs, clipboard e comando de terminal funcionam ou apresentam fallback |
| Sem backend durante detecção | Erro explícito; opção de nova verificação; projeto não é apagado |
| CLI instalada após Nexus iniciar | Verificar novamente refaz descoberta |
| Composer/Flow ocultos | Nenhum recurso experimental reaparece por conta da configuração de provider |

## Comportamentos intencionais

- Detectar instalação **não** cadastra perfil nem importa credenciais.
- A abertura do terminal não executa o comando de login automaticamente. O
  usuário permanece no controle da autenticação oficial.
- O estado `authenticated` retornado pelo backend determina a confirmação,
  mesmo se a cota estiver indisponível, desconhecida ou esgotada.
- Nenhum fluxo usa a API de projeção do HUD como fonte de autenticação.
- A autenticação de providers externos, o teste de terminal real e a
  certificação da plataforma devem ocorrer no dispositivo-alvo.

## Pendência

Build, lint, testes e smoke interativo de Windows/Linux precisam ser
executados antes de declarar GO. Esta alteração no GitHub não é, por si só,
evidência de homologação nativa.
