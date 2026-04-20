package telemetry

import "testing"

func TestTelemetryStress(t *testing.T) {
	t.Skip("Skipping - requires running server at localhost:8081")
}

func BenchmarkTelemetryPing(b *testing.B) {
	b.Skip("Skipping - requires running server at localhost:8081")
}
