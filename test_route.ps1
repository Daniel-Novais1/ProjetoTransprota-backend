# Script simplificado para simular uma rota longa
$baseUrl = "http://localhost:8080"
$deviceId = "test-bus-001"
$baseLat = -16.6869
$baseLng = -49.2648

# Usar token hardcoded temporariamente para teste (obtido via login manual)
# Em produção, isso seria obtido dinamicamente
$headers = @{
    Authorization = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NDUwNzI1MzEsInVzZXJfaWQiOiIxIiwidXNlcm5hbWUiOiJhZG1pbiJ9.5KfX9H3YJ8X3Z7Q2R1S8T9U0V1W2X3Y4Z5A6B7C8D9E0"
    "Content-Type" = "application/json"
}

# Gerar e enviar 30 pontos de telemetria
Write-Host "Enviando 30 pontos de telemetria..."

for ($i = 0; $i -lt 30; $i++) {
    # Variação pequena (pontos muito próximos - < 10m)
    $lat = $baseLat + ($i * 0.00001)
    $lng = $baseLng + ($i * 0.00001)
    
    # Para alguns pontos, fazer variação maior para testar quando NÃO decima
    if ($i % 10 -eq 0) {
        $lat = $baseLat + ($i * 0.0001)
        $lng = $baseLng + ($i * 0.0001)
    }
    
    $speed = 30 + (Get-Random -Minimum 0 -Maximum 20)
    $heading = Get-Random -Minimum 0 -Maximum 360
    
    $telemetry = @{
        device_id = $deviceId
        lat = [math]::Round($lat, 6)
        lng = [math]::Round($lng, 6)
        speed = $speed
        heading = $heading
        accuracy = 5.0
        transport_mode = "bus"
        route_id = "001"
        battery_level = 80
        recorded_at = (Get-Date).AddSeconds(-$i).ToString("o")
    } | ConvertTo-Json
    
    try {
        Invoke-WebRequest -Uri "$baseUrl/api/v1/telemetry/gps" -Method POST -Body $telemetry -Headers $headers -TimeoutSec 5 | Out-Null
        Write-Host "Ponto $i enviado: lat=$lat, lng=$lng"
    }
    catch {
        Write-Host "Erro ao enviar ponto $i"
    }
    
    Start-Sleep -Milliseconds 100
}

Write-Host ""
Write-Host "=================================================="
Write-Host "Rota simulada com sucesso!"
Write-Host "Device ID: $deviceId"
Write-Host "Total de pontos: 30"
Write-Host "Acesse http://localhost:5173/history para visualizar"
Write-Host "=================================================="
