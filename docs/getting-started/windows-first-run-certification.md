# Certificacao Windows: primeiro uso (release/v1.0.0)

O preflight automatizado verifica o codigo do **SHA exato** sem modificar perfis, credenciais, instalacoes externas ou o PATH permanente do usuario.

## Comando no Windows AMD64

No checkout limpo da branch release/v1.0.0, com Go 1.25.14 e Bun 1.3.9:

```powershell
git fetch origin release/v1.0.0
git switch release/v1.0.0
git pull --ff-only origin release/v1.0.0
$sha = (git rev-parse HEAD).Trim()
powershell -NoProfile -ExecutionPolicy Bypass -File .scriptscertify-windows-first-run.ps1 -ExpectedSha $sha
if ($LASTEXITCODE -ne 0) { throw "Windows first-run automated gate failed" }
```

O comando gera WINDOWS_FIRST_RUN_SUMMARY.json e logs locais sob o diretorio temporario. O JSON nao inclui identificadores de conta, credenciais ou transcricoes. Os logs brutos de private-logs devem ser revisados e sanitizados antes de qualquer upload.

## O que o preflight verifica

- Versoes de Go e Bun iguais as usadas no CI principal
- Testes de instalador/desinstalador com fixtures e PowerShell 5.1
- Regressao de diretorios invalidos, ConPTY/terminal, frontend typecheck e testes de UI
- Compilacao de CLI a partir do checkout
- Descoberta read-only de provedores em ambiente isolado
- Cadastro de pasta real no Windows com espacos e Unicode, persistencia entre processos e recusa de caminhos inexistentes/arquivos

Um gate que imprime erro ou sai com codigo diferente de zero fica FAIL; dependencias bloqueadas ficam BLOCKED. O processo termina com exit code 1 em qualquer FAIL ou BLOCKED.

## O que NAO esta automaticamente homologado

O script sempre marca **NOT_RUN** (e mantem NO_GO_WINDOWS) para:

1. Abertura real do Desktop Wails/WebView2, selecao de pasta e screenshot nativo
2. Navegacao prolongada no Codex com mouse, scroll, setas e menus via ConPTY
3. Quatro/cinco contas Codex autenticadas, isoladas e com quota independente
4. Instalacao por pacote assinado e ciclo de update/uninstall/rollback em ambiente descartavel

Esses gates exigem relatorio manual com SHA, resultado, procedimento, duracao e evidencia. Nao use prints sinteticos como prova de Desktop nativo.

## Distribuicao e consentimento

Uma instalacao via release assinada nao requer Go/Bun no computador do usuario. Na instalacao a partir do codigo, se faltarem compiladores, o instalador explica o requisito sem instalar ferramentas externas por conta propria. Apenas o parametro explicito `-InstallPrerequisites` autoriza o instalador a tentar instalar Go, Bun ou WebView2. `-Yes` sozinho **nao autoriza instalacao de dependencias externas**.

Nao publique v1.0.0 estavel enquanto o SHA do pacote publicado nao coincidir com o codigo validado e todos os gates nativos obrigatorios nao tiverem evidencias.
