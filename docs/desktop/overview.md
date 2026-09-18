# Desktop

O Desktop empacota o Core e a Web em um shell Wails nativo. Seu valor é integrar
lifecycle de processos, filesystem, notificações, deep links e seletores nativos
quando cada capacidade existe e está comprovada no sistema operacional.

O Nexus Desktop é a superfície nativa para operar o mesmo produto local. Seu
suporte deve ser lido junto da [matriz de plataformas](../operations/platform-support.md): compilar um binário não prova que todos os recursos nativos funcionam.

Use Desktop quando precisar da integração local fornecida pelo build nativo,
como processos, filesystem e terminal da plataforma. Use Web para controle no
navegador e CLI para automação ou operação headless.

O caminho Direct continua sendo o mesmo: projeto → AI Session → provider →
terminal. Maestro não é requisito para abrir o Desktop. A presença do código
não prova suporte; consulte a matriz antes de publicar uma promessa.
