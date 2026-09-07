# Terminais

Na Web, o terminal usa xterm para exibir uma sessão real. No Desktop/CLI, o backend usa o transporte nativo disponível na plataforma, como PTY Unix ou ConPTY/Named Pipe no Windows quando comprovado.

O Nexus mantém lifecycle, attach/detach, resize e controle de escrita. Nunca compartilhe tokens exibidos acidentalmente no terminal.
