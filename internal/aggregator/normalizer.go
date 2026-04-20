package aggregator

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
)

const (
	TimestampFormat = "2006-01-02T15:04:05Z07:00"
)

// NormalizedData formato padronizado interno (tid, dh, lat, long)
type NormalizedData struct {
	TID  string  `json:"tid"`  // Terminal/Route ID
	DH   string  `json:"dh"`   // Data/Hora formatada
	Lat  float64 `json:"lat"`  // Latitude
	Long float64 `json:"long"` // Longitude
	Hdg  float64 `json:"hdg"`  // Heading (opcional)
	Spd  float64 `json:"spd"`  // Speed (opcional)
	Ocp  string  `json:"ocp"`  // Occupancy (opcional)
	Src  string  `json:"src"`  // Source (rmtc/collective/cache)
	Cnf  float64 `json:"cnf"`  // Confidence (0-1)
}

// RMTCResponse formato de resposta da API RMTC (simulado)
type RMTCResponse struct {
	VehicleID string  `json:"vehicle_id"`
	RouteID   string  `json:"route_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Heading   float64 `json:"heading"`
	Speed     float64 `json:"speed"`
	Occupancy string  `json:"occupancy"`
	Timestamp string  `json:"timestamp"`
}

// CollectiveResponse formato de resposta da Inteligência Coletiva
type CollectiveResponse struct {
	DeviceHash   string  `json:"device_hash"`
	RouteID      string  `json:"route_id"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Heading      float64 `json:"heading"`
	Speed        float64 `json:"speed"`
	Contributors int     `json:"contributors"`
	Confidence   float64 `json:"confidence"`
	Timestamp    int64   `json:"timestamp"`
}

// Normalizer converte JSON externo para formato padronizado
type Normalizer struct{}

// NewNormalizer cria um novo normalizador
func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

// NormalizeFromRMTC converte resposta RMTC para formato padronizado
func (n *Normalizer) NormalizeFromRMTC(raw []byte) (*NormalizedData, error) {
	var rmtc RMTCResponse
	if err := json.Unmarshal(raw, &rmtc); err != nil {
		logger.Error("Aggregator", "Failed to unmarshal RMTC response: %v", err)
		return nil, fmt.Errorf("failed to parse RMTC response: %w", err)
	}

	// Converter timestamp
	timestamp, err := time.Parse(time.RFC3339, rmtc.Timestamp)
	if err != nil {
		timestamp = time.Now()
	}

	return &NormalizedData{
		TID:  rmtc.RouteID,
		DH:   timestamp.Format(TimestampFormat),
		Lat:  rmtc.Latitude,
		Long: rmtc.Longitude,
		Hdg:  rmtc.Heading,
		Spd:  rmtc.Speed,
		Ocp:  rmtc.Occupancy,
		Src:  "rmtc",
		Cnf:  1.0,
	}, nil
}

// NormalizeFromRMTCReader converte resposta RMTC de io.Reader para formato padronizado
func (n *Normalizer) NormalizeFromRMTCReader(reader io.Reader) (*NormalizedData, error) {
	var rmtc RMTCResponse
	if err := json.NewDecoder(reader).Decode(&rmtc); err != nil {
		logger.Error("Aggregator", "Failed to decode RMTC response: %v", err)
		return nil, fmt.Errorf("failed to parse RMTC response: %w", err)
	}

	// Converter timestamp
	timestamp, err := time.Parse(time.RFC3339, rmtc.Timestamp)
	if err != nil {
		timestamp = time.Now()
	}

	return &NormalizedData{
		TID:  rmtc.RouteID,
		DH:   timestamp.Format(TimestampFormat),
		Lat:  rmtc.Latitude,
		Long: rmtc.Longitude,
		Hdg:  rmtc.Heading,
		Spd:  rmtc.Speed,
		Ocp:  rmtc.Occupancy,
		Src:  "rmtc",
		Cnf:  1.0,
	}, nil
}

// NormalizeFromCollective converte resposta de Inteligência Coletiva para formato padronizado
func (n *Normalizer) NormalizeFromCollective(raw []byte) (*NormalizedData, error) {
	var coll CollectiveResponse
	if err := json.Unmarshal(raw, &coll); err != nil {
		logger.Error("Aggregator", "Failed to unmarshal Collective response: %v", err)
		return nil, fmt.Errorf("failed to parse Collective response: %w", err)
	}

	timestamp := time.Unix(coll.Timestamp, 0)

	return &NormalizedData{
		TID:  coll.RouteID,
		DH:   timestamp.Format(TimestampFormat),
		Lat:  coll.Latitude,
		Long: coll.Longitude,
		Hdg:  coll.Heading,
		Spd:  coll.Speed,
		Ocp:  "unknown",
		Src:  "collective",
		Cnf:  coll.Confidence,
	}, nil
}

// NormalizeFromBusData converte BusData interno para formato padronizado
func (n *Normalizer) NormalizeFromBusData(data *BusData) *NormalizedData {
	return &NormalizedData{
		TID:  data.RouteID,
		DH:   data.LastUpdate.Format(TimestampFormat),
		Lat:  data.Latitude,
		Long: data.Longitude,
		Hdg:  data.Heading,
		Spd:  data.Speed,
		Ocp:  data.Occupancy,
		Src:  string(data.Source),
		Cnf:  data.Confidence,
	}
}

// NormalizeFromJSON detecta automaticamente o formato e normaliza
func (n *Normalizer) NormalizeFromJSON(raw []byte, source DataSource) (*NormalizedData, error) {
	switch source {
	case DataSourceRMTC:
		return n.NormalizeFromRMTC(raw)
	case DataSourceCollective:
		return n.NormalizeFromCollective(raw)
	default:
		return nil, fmt.Errorf("unknown source: %s", source)
	}
}

// BatchNormalize normaliza múltiplos dados em lote
func (n *Normalizer) BatchNormalize(rawList [][]byte, source DataSource) ([]*NormalizedData, error) {
	results := make([]*NormalizedData, 0, len(rawList))

	for _, raw := range rawList {
		normalized, err := n.NormalizeFromJSON(raw, source)
		if err != nil {
			logger.Warn("Aggregator", "Failed to normalize batch item: %v", err)
			continue
		}
		results = append(results, normalized)
	}

	return results, nil
}

// ToJSON converte NormalizedData para JSON
func (n *NormalizedData) ToJSON() ([]byte, error) {
	return json.Marshal(n)
}

// FromJSON cria NormalizedData a partir de JSON
func FromJSON(data []byte) (*NormalizedData, error) {
	var normalized NormalizedData
	if err := json.Unmarshal(data, &normalized); err != nil {
		return nil, err
	}
	return &normalized, nil
}
