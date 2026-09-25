# Nexus HUD — Cross-Platform Completion

## Regra de produto

O HUD é cross-platform por definição. Depois do M1, a equipe não deve continuar
expandindo modos, painéis ou fontes de dados enquanto Windows e macOS estiverem
atrás do Linux na infraestrutura nativa.

Uma capacidade nativa só fica `DONE` quando o mesmo contrato foi:

1. implementado para Linux, Windows e macOS, ou explicitamente degradado por
   limitação documentada do compositor/OS;
2. compilado e executado em runner nativo no mesmo SHA;
3. coberto por smoke/evidência de runtime;
4. refletido na capability matrix sem claims inferidos.

## Origem da implementação

A implementação nasceu na branch histórica `feat/hud-cross-platform-completion`
e foi integrada à linha `v1.0.0-beta.3`. A branch não é mais uma linha de
produto separada.

Primeiro lote integrado:

- autostart opt-in com adapters Linux, Windows e macOS;
- promoção do artefato `nexus-hud` como tipo próprio no update manifest;
- distribuição Linux do HUD sem colisão com a chave de CLI;
- Windows/macOS executados como gates nativos desta própria branch;
- smoke de lançamento do HUD nos dois sistemas;
- capabilities continuam fail-closed para tray e multi-monitor/DPI.

Lotes seguintes, na linha v1 beta:

- tray/menu bar com Show/Hide/Quit e recovery seguro;
- auto-hide somente depois do recovery via tray;
- enumeração de displays com ID estável, work area e scale factor;
- restauração por display e clamp por work area;
- DPI/Retina e mudança dinâmica de monitor;
- comportamento X11/Wayland explicitamente capability-detected;
- empacotamento/manifesto para Windows e macOS depois da certificação.

## Definition of Done cross-platform

`NEXUS_HUD_CROSS_PLATFORM` só pode sair de `NO_GO` quando:

- Linux amd64, Windows amd64 e macOS arm64 passarem no mesmo SHA;
- tray/recovery, autostart, always-on-top e window restore tiverem evidência
  nativa nos três targets;
- multi-monitor/DPI tiver comportamento certificado ou degradação explícita;
- os artefatos HUD tiverem checksums, proveniência e update-manifest sem
  colisões de chave;
- documentação e capability flags corresponderem exatamente à evidência.


## Runners nativos do gate

Para evitar que a paridade dependa de máquinas self-hosted estarem ligadas, a
frente cross-platform executa os gates obrigatórios em runners GitHub-hosted:

- Windows x64: `windows-2025`;
- macOS arm64: `macos-15`.

O certificado registra o runner utilizado. O requisito continua sendo
execução nativa no SHA exato; cross-compilation não conta como evidência.
