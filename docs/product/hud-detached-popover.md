# HUD nativo: janela de popup independente (experimento)

**Estado:** implementação opt-in em `feat/hud-detached-popover`.
Não declarar homologado ou substituir a cápsula atual antes de testes
interativos no Windows, macOS e Linux X11.

## Objetivo

O Nexus HUD mantém uma cápsula pequena (por padrão 220 x 60 DIPs)
e exibe os detalhes em uma segunda janela Wails v3, inicialmente
oculta (por padrão 334 x 496 DIPs). O popup tem WebView/React próprios,
mas usa o mesmo Core autenticado; não duplica API, sessão de quotas,
probes de providers ou mecanismos de roteamento.

### Ativação reversível

Configure **somente no processo de teste**:

```bash
NEXUS_HUD_DETACHED_POPOVER=1 ./cmd/nexus-hud/build/bin/nexus-hud
```

Windows PowerShell, em sessão de homologação:

```powershell
$env:NEXUS_HUD_DETACHED_POPOVER = '1'
.\cmd\nexus-hud\build\bin\nexus-hud.exe
```

O modo continua **desligado por padrão**. Remover a variável restaura
a expansão legada na mesma janela. Linux Wayland utiliza sempre o modo
legado: sem posicionamento global confiável, o popup separado poderia
aparecer fora da área visível.

## Componentes

- `cmd/nexus-hud/main.go` cria a cápsula habitual e, somente quando
  opt-in, uma segunda janela `Nexus HUD Popup`, com rota
  `/?nativeHud=1&hudView=popover` e `Hidden=true`.
- `cmd/nexus-hud/window_options.go` compartilha o contrato de
  transparência, moldura, sombras e escala das opções Wails.
- `internal/desktop/hud_detached.go` ancora o popup acima da
  cápsula, ou abaixo se não houver espaço, e limita seu tamanho à
  work area. A cápsula nunca recebe as dimensões do painel.
- `HUDController.SetHUDExpanded` continua o único contrato
  de abertura/fechamento. Ele redimensiona apenas no modo legado;
  no modo separado, mostra/oculta a segunda janela.
- `NexusHud.tsx` renderiza apenas a cápsula na primeira WebView e
  apenas o painel na segunda. `GetHUDExpanded` sincroniza fechamento
  por Escape, perda de foco e botões, evitando estados presos.
- O tray continua associado à cápsula; a janela popup não substitui
  o processo ou os dados existentes.

## Gates obrigatórios

1. Build e testes unitários Go: `./internal/desktop/...` e
   `./cmd/nexus-hud`; lint/typecheck/format e testes Web.
2. Windows/macOS/Linux X11: clicar 20 vezes para abrir/fechar;
   medir que **a cápsula permanece compacta** e o popup é outra
   janela visível, com título/handle próprios.
3. Arrastar a cápsula, usar presets Compacto/Confortável/Grande,
   e abrir novamente; popup deve surgir perto da cápsula.
4. Verificar o comportamento da perda de foco, Escape, tray,
   Core offline, kill switch, DPI/monitores e reinicialização.
5. Registrar screenshots reais das duas janelas, sombreamento
   e bloqueio de cliques nas regiões transparentes. Dois
   retângulos transparentes não significam click-through perfeito.
6. Em Linux Wayland, verificar que a variável não força o modo
   separado; manter a cápsula legada.

No Windows, o script nativo já possui um gate interativo específico:

```powershell
$sha = (git rev-parse HEAD).Trim()
.\scripts\homologate-hud-native-windows.ps1 `
  -HudExe (Resolve-Path '.\cmd\nexus-hud\build\bin\nexus-hud.exe').Path `
  -EvidenceDirectory (Join-Path $PWD 'hud-evidence/windows-detached') `
  -ExpectedSha $sha -Cycles 20 -Detached
```

Esse gate exige que a cápsula permaneça compacta, localiza a segunda
janela pelo seu título nativo, abre/fecha 20 vezes e captura a janela real.
Ainda requer sessão desktop e revisão humana da transparência.

Os scripts Linux/macOS `homologate-hud-native-*.sh` continuam cobrindo
a geometria **legada**, não certificam automaticamente o popup separado;
adaptações equivalentes e execução nativa permanecem obrigatórias.

**Critério de merge:** sem GO nativo, manter PR em Draft. A
implementação não corrige, por si só, click-through de pixels
transparentes no Windows/WebView2 ou GTK/Wayland.
