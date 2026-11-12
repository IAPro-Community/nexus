# ADR: Pareamento Remoto por Código de Acesso (alternativa ao QR)

## Status
Aceito e Implementado (parcial — falta E2E em túnel público com dois dispositivos físicos)

## Contexto
O acesso remoto do Nexus usa Cloudflare Quick Tunnel e um convite de uso único
(`POST /api/v1/tunnel/invite`) cujo token de 64 hex é embutido na URL
`https://<túnel>/#nexus_bootstrap=<token>` renderizada como QR Code
(`RemoteAccessTab`). O pareamento exige escanear ou copiar o link inteiro.
Em cenários reais (celular sem câmera disponível, compartilhamento verbal,
acessibilidade), o maestro precisa de uma alternativa textual curta: gerar um
código no desktop e digitá-lo no dispositivo remoto.

## Decisão
Adotar **código de acesso Crockford Base32 de 8 caracteres**, emitido junto com
cada convite de túnel e trocado por sessão via `POST /api/v1/auth/pair`.

### Princípios Chave
1. **O código herda o convite, não substitui o modelo**: mesmo vínculo ao host
   do túnel, mesmo TTL (15 min), mesmo uso único e mesma revogação quando o
   túnel para ou o Nexus reinicia (`SetTunnelActive` zera `remoteInvitations`).
2. **Entropia suficiente**: 32^8 ≈ 2^40 candidatos; alfabeto sem I/L/O/U
   evita ambiguidade visual e entrada é normalizada (I/L→1, O→0, caixa e
   pontuação ignoradas) para tolerar digitação.
3. **Guarda anti-força-bruta global**: 10 falhas consecutivas revogam todos os
   convites pendentes; o emissor precisa gerar novo convite. Probabilidade de
   adivinhação permanece desprezível (≤10/2^40 por janela).
4. **Mesma sessão, mesma política**: o pareamento usa `newSessionLocked` e
   produz sessão idêntica ao bootstrap do QR — cookie HttpOnly/Secure, CSRF,
   `NonPersistable` para exposição remota; nunca persistida em disco.
5. **Sem novo canal de credencial**: o código nunca aparece em logs, URLs ou
   armazenamento; comparação em tempo constante (`subtle.ConstantTimeCompare`);
   GET nunca emite nem revela credencial (mesma regra do convite).
6. **UI dedicada**: `AccessCodeGate` na tela não autenticada (só em hosts não
   loopback) e exibição/cópia do código em `RemoteAccessTab`; i18n PT/EN/ES e
   tokens de design, sem exceções.

## Alternativas Consideradas
- **Somente QR (status quo)**: rejeitado — inacessível sem câmera/segunda tela.
- **Código numérico de 6 dígitos**: rejeitado — espaço de 10^6 com TTL longo
  exige rate-limit por IP que o túnel público não fornece de forma confiável.
- **Token curto derivado do token completo**: rejeitado — reduzir o token
  enfraqueceria o canal QR existente; os dois segredos são independentes.

## Consequências
- Contrato V1: `remote-companion` ganha terceiro critério de aceite (48
  capacidades preservadas; verificador continua exigindo ≥2 critérios).
- Evidência local completa em testes Go e Vitest; aceite `verified` exige E2E
  real em túnel público com dois dispositivos físicos no SHA distribuído.
- Revisão de segurança futura deve reavaliar o limite de 10 falhas e o TTL de
  15 min quando houver telemetria de uso real.
