# Validação do Companion 2D e Guardian

Veredito obrigatório: NO-GO — COMPANION_2D_NOT_INTEGRATED.

## Conferência de integração

No código efetivamente compilado no HEAD bf6adf0c80ac3531d7291844512a14a017ab0002, o HUD usa GuardianMascot.tsx, guardianModel e NexPixelSprite. Não existem NexusGuardian.tsx, companionModel nem o atlas/animações contextuais requeridos pelo Companion 2D da PR #55. O caminho real do HUD renderiza o Guardian anterior; identificadores semelhantes não foram tratados como equivalentes ao Companion.

A consulta viva ao GitHub em 2026-10-23 encontrou a PR #55 aberta e em draft, com base release/v1.0.0 e head feat/nexus-companion-v1.1. Ela não está integrada ao commit testado. A descrição da PR registra a necessidade de conciliação com o Guardian Mascot para evitar duas implementações paralelas. A versão 1.0.0 atual não contém o Companion novo.

Evidência de fonte: logs/companion-source-audit.txt e logs/pull-requests-final.txt. Referência: https://github.com/IAPro-Community/nexus/pull/55.

## Teste visual nativo

- A janela HUD nativa exibiu o Guardian pixel art antigo no XWayland: screenshots/native-hud-guardian-xwayland-bf6adf0c.png.
- A captura da configuração mostra as opções Barra, Mascote e Bandeja e a restrição de popover no Wayland: screenshots/native-hud-after-import-bf6adf0c.png.
- A captura prova renderização do Guardian antigo e abertura dos controles. Ela não prova renderização ou interação do Nexus Companion 2D.
- Não foram validados sprites, animações idle/working/attention/critical/offline, cápsula de pendências, movimento reduzido, deduplicação do Attention Center, adiar/silenciar, click-through nem persistência específica do Companion: a implementação não está integrada.

## Conclusão

O Guardian antigo está renderizando. O Nexus Companion 2D obrigatório não está no binário homologado. Aplicar o resultado específico requerido: NO-GO — COMPANION_2D_NOT_INTEGRATED. Não aprovar nem substituir o Companion ausente pelo Guardian antigo.
