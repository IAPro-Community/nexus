# NEX — seis vistas em 256 px (referências, NÃO atlas 360 final)

Esta pasta versiona seis ilustrações do personagem escolhido: frente, duas inclinações discretas, dois 3/4 e costas. WebPs 256×256 criados com redução Lanczos da fonte 1254×1254 e **WebP lossless**. Os arquivos têm **fundo navy opaco**, não alpha, e não são sprites transparentes finalizados para HUD.

Vistas geradas independentemente não asseguram geometria ou oclusão contínua. Os ângulos nominais *shallow/three_quarter* não são medições precisas de yaw. Faltam perfis verdadeiros, três quartos traseiros e 16 vistas consistentes com pivô. Não conectar estes quadros ao atlas oficial 360°, não dar status APPROVED, nem substituir o master. A homepage permanece bloqueada.

`contact-sheet.webp` permite visualizar todas as vistas; `index.html` serve para QA local; `manifest.json` guarda hashes SHA-256, fontes e status. A fonte Hero original e o master pixel art seguem separados. Falta gerar transparência real, poses intermediárias e validar Compact 64/Standard 128 em HUD nativo. Ver `docs/design/nex/RELEASE_READINESS.json`.
