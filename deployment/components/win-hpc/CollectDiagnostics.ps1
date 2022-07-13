$ErrorActionPreference = "Stop"

$runIdPath = Join-Path $env:CONTAINER_SANDBOX_MOUNT_POINT "config\run_id"
$outputFolder = "\k\periscope-diagnostic-output"
$logsPath = "${outputFolder}\logs"

# Ensure the output directory exists
New-Item -ItemType Directory $outputFolder -Force

# For tracking contents of run_id file (and run diagnostics collection script when it changes)
$previousRunId = ""

while ($true) {
    $runId = Get-Content $runIdPath
    if ($runId -ne $previousRunId) {
        Write-Host "Collecting diagnostics for ${runId}"

        # The FileInfo containing the zip file is the last output from the script
        $outputs = (& "C:\k\Debug\collect-windows-logs.ps1")
        $logsZipFileInfo = $outputs[$outputs.Length - 1]

        # Replace any existing log files with the unzipped content
        Remove-Item -LiteralPath "${outputFolder}\*" -Force -Recurse
        Expand-Archive -Path $logsZipFileInfo.FullName -Force -DestinationPath $logsPath

        # Create an empty file to notify any watchers that log collection is completed for this run,
        # and update previous-run tracker to avoid repeated re-runs.
        New-Item "${outputFolder}\${runId}"
        $previousRunId = $runId
    }

    Start-Sleep -Seconds 10
}
