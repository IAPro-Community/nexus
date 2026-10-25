# Ambiente Linux da homologação

Data: 2026-10-23. Arquitetura de destino: Linux AMD64. SHA-base: bf6adf0c80ac3531d7291844512a14a017ab0002. O checkout local contém alterações não commitadas; ver seção de proveniência no relatório principal.

## Sistema e sessão

- Ubuntu 25.10, Questing Quokka; kernel 6.17.0-23-generic.
- x86_64 / AMD64, 8 CPUs online; CPU virtualizada AMD EPYC 7313.
- GNOME Ubuntu em Wayland, sessão wayland-0. DISPLAY :0 corresponde a XWayland; não havia uma sessão X11 independente.
- Monitor único VMware SVGA II, 1920×1080, 59.96 Hz, escala GNOME 1.0. Não havia configuração para comparar múltiplos monitores ou escalas.
- GPU VMware virtual, sem aceleração 3D habilitada. Wails/WebKit registraram “VMware: No 3D enabled”; glxinfo não estava disponível.
- Memória: 60 GiB totais, aproximadamente 23 GiB disponíveis durante a coleta inicial; swap 4 GiB, cerca de 370 MiB livres. Carga do host variou durante os testes.

## Toolchain e bibliotecas

- Go inicialmente ativo: go1.25.14. O toolchain go1.26.10 exigido pelo fluxo Linux do projeto foi baixado pelo mecanismo de toolchain do Go para o cache do usuário, sem instalação global; a verificação final e o rebuild usam GOTOOLCHAIN=go1.26.10.
- Bun 1.4.2; Node.js 22.17.0.
- GCC 15.2.0-4ubuntu4; GNU Make 4.4.1.
- Wails v3.0.0-beta.25, conforme go.mod e web/package.json.
- GTK 3.24.50; WebKitGTK 4.1 2.52.3; JavaScriptCoreGTK 2.52.3; libsoup 3.6.5.
- O Desktop e o HUD foram compilados com a tag GTK 3. ldd não encontrou bibliotecas nativas ausentes; GTK, WebKitGTK, JavaScriptCoreGTK e libsoup estão vinculados.

## Permissões e evidência nativa

A sessão do usuário permitiu iniciar Wails Desktop, o serviço Web e o HUD. A janela principal e a janela HUD foram observadas como aplicações nativas; as capturas X11 foram obtidas através do XWayland. O Desktop também foi iniciado na sessão Wayland, mas as limitações de captura/interação nativa do popover nesse compositor permanecem registradas em HUD_VALIDATION.md.

Capturas e hashes estão em screenshots/ e logs/screenshots.sha256. Os arquivos XWD intermediários foram removidos após confirmar que as capturas PNG correspondentes existiam.
