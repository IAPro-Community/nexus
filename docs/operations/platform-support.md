# Suporte de plataformas

Esta página separa build de runtime. O estado abaixo é uma política
documental: uma linha só deve ser marcada como verificada quando houver teste
ou CI nativo correspondente.

| Área | Linux | Windows | macOS |
| --- | --- | --- | --- |
| Core/CLI build | Consulte CI | Consulte CI | Consulte CI |
| Web | Consulte CI | Consulte CI | Consulte CI |
| Desktop Wails | Consulte CI nativo | Consulte CI nativo | Consulte CI nativo |
| Terminal nativo | PTY/contrato do host | ConPTY/PowerShell | contrato do host |
| Installer/update | Consulte release | Consulte release | Consulte release |

Não trate cross-compilation como prova de runtime. Para diagnóstico, consulte
[troubleshooting](troubleshooting.md) e os relatórios de validação em `DEV/`.
