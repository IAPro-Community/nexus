# NEX – Hover 01, atlas frontal em 256 px

16 quadros RGBA de 256×256 gerados a partir do PNG frontal originalmente aprovado (1254×1254, com alfa), sem redesenho facial. Loop de 2 s / 8 fps, deslocamento vertical -3..+3px e horizontal -1..+1px após redução única com Lanczos. Não são 16 ângulos de rotação: todos os quadros usam **a mesma vista frontal**.

O movimento nesta primeira iteração desloca o personagem como um todo, incluindo a órbita. Não substitui o rig independente dos quatro módulos nem o giro 360°. O master original permanece intacto e a versão do HUD público não é substituída automaticamente.

`nex-hover-256-atlas.png` é o spritesheet 4×4; WebP é a mesma arte comprimida lossless; `frames/` contém os 16 PNGs individuais. A prévia HTML permite testar reprodução, pausa e tamanho nativo. Respeite `prefers-reduced-motion`.
