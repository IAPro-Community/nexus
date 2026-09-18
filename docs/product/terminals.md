# Terminais

Na Web, o terminal usa xterm para exibir uma sessão supervisionada. A captura
visual publicada usa um shell de projeto em workspace sintético isolado e só é
marcada como evidência quando um comando-marcador retorna no xterm sem erro,
desconexão ou recuperação.

No Desktop/CLI, o backend usa o transporte nativo disponível na plataforma,
como PTY Unix ou ConPTY/Named Pipe no Windows, somente quando a matriz de
suporte e os testes nativos correspondentes comprovarem esse caminho.

O Nexus mantém lifecycle, attach/detach, resize e controle de escrita. Nunca compartilhe tokens exibidos acidentalmente no terminal.
