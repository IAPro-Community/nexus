# Varredura heurística de segredos nos logs da homologação

Data: 2026-10-24

Arquivos de texto UTF-8 examinados sob logs/: 65.

Padrões procurados: famílias comuns de API keys (OpenAI/GitHub/Google), JWT e cabeçalho Bearer longo. A execução conta apenas arquivos com correspondência e nunca registra os valores encontrados.

Arquivos com padrão API key: 0.
Arquivos com padrão JWT: 0.
Arquivos com padrão Bearer: 0.
Resultado: nenhum padrão encontrado.

Limite: heurística por padrões conhecidos; não é auditoria completa de todos os formatos ou dados do host.
