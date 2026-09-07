# Desktop

O Nexus Desktop é a superfície nativa baseada em Wails para operar o mesmo
produto local. Seu suporte deve ser lido junto da [matriz de plataformas](../operations/platform-support.md): compilar um binário não prova que todos os recursos nativos funcionam.

Use Desktop quando precisar da integração local fornecida pelo build nativo,
como processos, filesystem e terminal da plataforma. Use Web para controle no
navegador e CLI para automação ou operação headless.

O caminho Direct continua sendo o mesmo: projeto → AI Session → provider →
terminal. Maestro não é requisito para abrir o Desktop.
