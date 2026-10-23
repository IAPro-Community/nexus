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

## Attach ao Web Core e assets estáticos (beta.10)

O handler de attach em `internal/control/web/desktop_attach.go` deve atender
assets reais de JavaScript e CSS antes de aplicar o fallback SPA. Quando a
requisição de bundle recebe `index.html` no lugar do arquivo solicitado, o
Desktop pode mostrar uma janela vazia/preta apesar de o processo Core estar
ativo. O teste `internal/control/web/desktop_attach_test.go` exercita a
entrega de `bundle.js`, `bundle.css` e uma rota SPA.

A correção é evidência do caminho HTTP, não substitui a abertura de um Desktop
Wails no Windows e uma captura da janela com WebView2 no **mesmo SHA**.
Consulte [suporte nativo](../operations/platform-support.md) e
[status da release](../release-status.md).
