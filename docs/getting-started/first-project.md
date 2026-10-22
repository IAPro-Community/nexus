# Primeiro projeto e primeira sessão

Pré-requisito: Nexus instalado — [instalação nativa](installation.md), [pacotes](packages.md) ou [Docker](docker-compat.md).

## Primeiro uso guiado

1. Abra o Nexus. Antes de cadastrar o primeiro projeto, a tela **Ambiente de execução** verifica as CLIs instaladas e o estado de registro sem alterar contas.
2. Se não houver CLI instalada, escolha as instruções oficiais de Codex, Claude Code ou OpenCode. Instale a ferramenta e clique em **Verificar novamente**.
3. Se encontrar uma CLI instalada sem perfil, use **Adicionar conta**. O registro é explícito; presença no PATH não significa autenticação.
4. Se a conta estiver pendente ou expirada, selecione **Concluir login**; não crie outra conta por engano.
5. Execute o comando de login oficial exibido no assistente, depois clique em **Verificar**. O Nexus só confirma a etapa quando receber autenticação verdadeira do backend. Disponibilidade e cota são verificadas separadamente.
6. Adicione a pasta do projeto. Use **+ New → AI Session**, escolha a conta conectada e execute uma tarefa simples e reversível.

Sem CLI, o Nexus ainda permite criar o projeto e usar o **Project Shell**. Em projetos existentes, a **Visão geral** informa quando uma conta precisa de configuração e oferece **Contas de IA**. Também há um caminho em **Primeiros passos → Configurar contas de IA**.

## HUD desligado

O HUD é um painel opcional de observabilidade. Detecção de CLIs, cadastro de perfis, login e atualização de recursos usam as APIs de providers e recursos, independentes do HUD. Ele não precisa estar habilitado para gerenciar contas ou iniciar sessões.

Se o modal de login oferece **Abrir terminal**, o Nexus abre o Project Shell e tenta copiar o comando `nexus login <provider> <profile>` para a área de transferência; o usuário ainda precisa executá-lo no terminal. Quando não há projeto, o assistente oferece o comando para copiar e executar externamente. O Nexus nunca importa credenciais ou instala ferramentas de terceiros silenciosamente.

Na distribuição pública da 1.0.0, Composer e Flow continuam ocultos atrás de uma flag de build. O uso direto de providers não depende deles.

## Se a sessão parar

Distinga a identidade do Agent da geração concreta de Runtime. Um runtime encerrado pode ser recuperável, mas não é automaticamente a mesma sessão do provider. A UI deve mostrar a continuidade real, ou um estado não verificado.
