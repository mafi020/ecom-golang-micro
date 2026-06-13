$ports = @(50050, 50051, 50052, 50053, 50054)
foreach ($port in $ports) {
    $pids = netstat -ano | Select-String ":$port\s" | ForEach-Object {
        ($_ -split '\s+')[-1]
    } | Sort-Object -Unique

    foreach ($p in $pids) {
        if ($p -match '^\d+$') {
            Write-Host "Killing PID $p on port $port"
            taskkill /PID $p /F 2>$null
        }
    }
}