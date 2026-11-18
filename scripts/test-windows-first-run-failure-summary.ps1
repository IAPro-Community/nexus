# Verify public diagnostics contain test names only, with synthetic secrets.
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'windows-first-run-failure-summary.ps1')

$raw = @'
C:\Users\alice\secret-project\source.go
NEXUS_SESSION_TOKEN=SECRET_TOKEN_WOULD_LEAK
--- FAIL: TestOutputFlushOnClose (0.04s)
--- FAIL: TestHostWriterHandoff (0.20s)
--- FAIL: TestHostWriterHandoff (0.20s)
FAIL github.com/IAPro-Community/nexus/internal/control/terminal
'@
$summary = Get-NexusGoFailureSummary -RawDiagnostics $raw
$expected = 'Go test failures: TestHostWriterHandoff, TestOutputFlushOnClose; see private log'
if ($summary -ne $expected) { throw 'Sanitizer did not emit stable test identifiers' }
if ($summary -match 'alice|secret|TOKEN|source\.go|Users') {
    throw 'Sanitizer emitted confidential test output'
}
if ((Get-NexusGoFailureSummary -RawDiagnostics 'panic: test timed out after 10m0s') -ne 'Go test timed out; see private log') {
    throw 'Sanitizer did not recognize timeouts'
}
if ((Get-NexusGoFailureSummary -RawDiagnostics 'FAIL') -ne 'Go test failed; see private log') {
    throw 'Sanitizer did not fail closed'
}
Write-Host '[PASS] Sanitized ConPTY diagnostics expose only test identifiers.'
