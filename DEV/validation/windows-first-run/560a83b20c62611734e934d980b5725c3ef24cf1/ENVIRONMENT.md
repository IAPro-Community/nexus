# Ambiente de Homologação — Nexus 1.0.0 (Windows)

- **Data da Homologação:** 2026-10-18 / 2026-10-19
- **Repositório:** `https://github.com/IAPro-Community/nexus`
- **Branch:** `release/v1.0.0`
- **Commit SHA:** `560a83b20c62611734e934d980b5725c3ef24cf1`
- **Status do Worktree:** Limpo (100% clean)

---

## 1. Sistema Operacional e Hardware

| Propriedade | Valor Detectado |
| :--- | :--- |
| **OS Name** | Microsoft Windows 11 Home Single Language |
| **OS Version** | 10.0.26200 N/A Build 26200 |
| **Arquitetura** | AMD64 (x86_64) |
| **ConPTY** | Disponível nativamente (Windows Build 26200 >= 17763) |
| **Microsoft Edge WebView2** | Instalado (`155.0.4283.62`) |
| **Processador** | AMD64 Family 25 Model 80 Stepping 0 AuthenticAMD ~3301 Mhz |

---

## 2. Shells e Runtimes

| Ferramenta | Caminho / Versão | Status |
| :--- | :--- | :--- |
| **Windows PowerShell** | `5.1.26100.9711` Desktop Edition | Nativo |
| **PowerShell Core (pwsh)** | `7.x / 8wekyb3d8bbwe\pwsh.exe` | Instalado via WindowsApps |
| **Go** | `go version go1.25.14 windows/amd64` | Instalado (diverge de `go1.26.9` exigido estritamente em `homologate-beta.ps1`) |
| **Bun** | `1.3.9` | Instalado e compatível com CI |
| **Node.js** | `v24.21.0` | Instalado |
| **Git** | `2.56.0.windows.2` | Instalado |

---

## 3. Ferramentas de Empacotamento e Assinatura

| Ferramenta | Status | Observação |
| :--- | :--- | :--- |
| **Wails CLI** | Não instalado no PATH | Necessário para build nativo empacotado Wails v3 |
| **NSIS (makensis)** | Não instalado no PATH | Necessário para compilar o instalador `nexus-setup-Windows_x86_64.exe` |
| **Signtool** | Não instalado no PATH | Necessário para assinatura Authenticode de executáveis Windows |
| **Chave Ed25519** | Verificada | Chave pública canônica `ee2e268394ce6a0634a9ea2f414ee831dd536d12f7e2b80107f94566e58aa334` |

---

## 4. Provedores de IA Locais Detectados

| Provedor | Binário Detectado | Versão | Status |
| :--- | :--- | :--- | :--- |
| **Codex** | `C:\Users\kiver\AppData\Local\Programs\OpenAI\Codex\bin\codex.exe` | `codex-cli 0.163.0` | Instalado / Autenticado no ambiente do usuário |
| **AGY** | `C:\Users\kiver\AppData\Local\agy\bin\agy.exe` | `1.4.0` | Instalado |
| **Gemini** | `C:\Program Files\nodejs\gemini` | `0.38.2` | Instalado |
| **OpenCode** | `C:\Program Files\nodejs\opencode` | `v2.0.28` | Instalado |
| **Claude** | Não detectado no PATH | N/A | Não instalado |
| **Cursor** | Não detectado no PATH | N/A | Não instalado |
