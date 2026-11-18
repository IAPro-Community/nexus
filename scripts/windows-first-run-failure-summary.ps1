# Report only identifiers defined in test source. Never expose raw stderr.
function Get-NexusGoFailureSummary {
    param([AllowEmptyString()][string]$RawDiagnostics = '')

    $testNames = @(
        [regex]::Matches($RawDiagnostics, '(?m)^[ \t]*--- FAIL:\s+(Test[A-Za-z_][A-Za-z0-9_]*)\b') |
            ForEach-Object { $_.Groups[1].Value } |
            Sort-Object -Unique |
            Select-Object -First 8
    )
    if ($testNames.Count -gt 0) {
        return ('Go test failures: ' + ($testNames -join ', ') + '; see private log')
    }
    if ($RawDiagnostics -match 'panic:\s+test timed out') {
        return 'Go test timed out; see private log'
    }
    return 'Go test failed; see private log'
}
