# IAPro Nexus — Aviso de privacidade (rascunho para validação)

> Esta página descreve princípios pretendidos; **não deve ser tratada como garantia técnica auditada** até a implementação de rede, diagnóstico, coleta e atualização ser verificada no SHA publicado. Revisar com o controlador de dados antes de distribuição pública.

## Dados locais e providers

O Nexus é projetado para operar localmente projetos, sessões e credenciais. **Providers, CLIs, serviços de túnel e integrações de terceiros podem transmitir dados de acordo com seus próprios termos**. Não prometa que todo conteúdo “nunca sai do dispositivo” sem inspecionar o provider e o recurso efetivamente usado.

## Analytics e diagnóstico

O produto prevê analytics opt-in e diagnóstico sob controle do usuário. Antes da publicação, demonstrar via testes e inspeção de tráfego que os defaults, event allowlists, sanitização e comandos de opt-out correspondem ao código da versão publicada. Não compartilhar tokens, caminhos sensíveis, projetos nem logs de terminal inadvertidamente.

## Atualização

Artefatos de atualização devem ter assinatura e hash verificáveis. Verificar e documentar dados enviados em checagens de update, download e autenticação de sessão. Ver [Segurança](../../SECURITY.md).

## Direitos e contato

Identificar o responsável pelo tratamento, a base legal aplicável e um canal de contato verificável **antes de tornar o repositório ou os binários públicos**.

## Licenciamento

A política do source está em [LICENSING.md](LICENSING.md). O uso de qualquer provider depende adicionalmente dos respectivos contratos.
