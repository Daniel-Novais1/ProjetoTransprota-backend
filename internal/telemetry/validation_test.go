package telemetry

import (
	"testing"
	"time"
)

// TestValidatePingTableDriven usa table-driven tests para validação de GPS ping
func TestValidatePingTableDriven(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		ping    *TelemetryPing
		wantErr bool
	}{
		{
			name: "Ping válido",
			ping: &TelemetryPing{
				DeviceID:     "test-device",
				Latitude:     -16.6869,
				Longitude:    -49.2648,
				Speed:        30.5,
				RecordedAt:   now,
				BatteryLevel: 85,
			},
			wantErr: false,
		},
		{
			name: "Latitude inválida - muito alta",
			ping: &TelemetryPing{
				DeviceID:     "test-device",
				Latitude:     999.0,
				Longitude:    -49.2648,
				Speed:        30.5,
				RecordedAt:   now,
				BatteryLevel: 85,
			},
			wantErr: true,
		},
		{
			name: "Latitude inválida - muito baixa",
			ping: &TelemetryPing{
				DeviceID:     "test-device",
				Latitude:     -999.0,
				Longitude:    -49.2648,
				Speed:        30.5,
				RecordedAt:   now,
				BatteryLevel: 85,
			},
			wantErr: true,
		},
		{
			name: "Longitude inválida - muito alta",
			ping: &TelemetryPing{
				DeviceID:     "test-device",
				Latitude:     -16.6869,
				Longitude:    999.0,
				Speed:        30.5,
				RecordedAt:   now,
				BatteryLevel: 85,
			},
			wantErr: true,
		},
		{
			name: "Longitude inválida - muito baixa",
			ping: &TelemetryPing{
				DeviceID:     "test-device",
				Latitude:     -16.6869,
				Longitude:    -999.0,
				Speed:        30.5,
				RecordedAt:   now,
				BatteryLevel: 85,
			},
			wantErr: true,
		},
		{
			name: "Velocidade negativa",
			ping: &TelemetryPing{
				DeviceID:     "test-device",
				Latitude:     -16.6869,
				Longitude:    -49.2648,
				Speed:        -10.0,
				RecordedAt:   now,
				BatteryLevel: 85,
			},
			wantErr: true,
		},
		{
			name: "Velocidade acima do limite",
			ping: &TelemetryPing{
				DeviceID:     "test-device",
				Latitude:     -16.6869,
				Longitude:    -49.2648,
				Speed:        200.0,
				RecordedAt:   now,
				BatteryLevel: 85,
			},
			wantErr: true,
		},
		{
			name: "Bateria acima do limite",
			ping: &TelemetryPing{
				DeviceID:     "test-device",
				Latitude:     -16.6869,
				Longitude:    -49.2648,
				Speed:        30.5,
				RecordedAt:   now,
				BatteryLevel: 150,
			},
			wantErr: true,
		},
		{
			name: "Bateria abaixo do limite",
			ping: &TelemetryPing{
				DeviceID:     "test-device",
				Latitude:     -16.6869,
				Longitude:    -49.2648,
				Speed:        30.5,
				RecordedAt:   now,
				BatteryLevel: -10,
			},
			wantErr: true,
		},
		{
			name: "Timestamp no futuro",
			ping: &TelemetryPing{
				DeviceID:     "test-device",
				Latitude:     -16.6869,
				Longitude:    -49.2648,
				Speed:        30.5,
				RecordedAt:   now.Add(1 * time.Hour),
				BatteryLevel: 85,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{}
			errs := c.validatePing(tt.ping)
			if (len(errs) > 0) != tt.wantErr {
				t.Errorf("validatePing() wantErr=%v, got %v", tt.wantErr, errs)
			}
		})
	}
}

// TestAnonymizeDeviceIDTableDriven testa anonimização de device ID
func TestAnonymizeDeviceIDTableDriven(t *testing.T) {
	tests := []struct {
		name     string
		deviceID string
		wantLen  int
	}{
		{
			name:     "Device ID normal",
			deviceID: "device-123",
			wantLen:  64, // SHA-256 hash
		},
		{
			name:     "Device ID vazio",
			deviceID: "",
			wantLen:  64,
		},
		{
			name:     "Device ID longo",
			deviceID: "very-long-device-id-with-many-characters-123456789",
			wantLen:  64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{}
			hash := c.anonymizeDeviceID(tt.deviceID)
			if len(hash) != tt.wantLen {
				t.Errorf("anonymizeDeviceID() hash length = %v, want %v", len(hash), tt.wantLen)
			}
		})
	}
}
