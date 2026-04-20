package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/redis/go-redis/v9"
)

const (
	MinVotesForConsensus = 3
	MaxOutlierDistanceM  = 100
	ConsensusTTL         = 2 * time.Minute
)

type UserReport struct {
	DeviceHash string
	Latitude   float64
	Longitude  float64
	Speed      float64
	Heading    float64
	Timestamp  time.Time
}

type BusConsensus struct {
	RouteID      string
	Latitude     float64
	Longitude    float64
	Speed        float64
	Heading      float64
	Contributors int
	Confidence   float64
	Timestamp    time.Time
}

type CollectiveIntelligence struct {
	rdb *redis.Client
	mu  sync.RWMutex
}

func NewCollectiveIntelligence(rdb *redis.Client) *CollectiveIntelligence {
	return &CollectiveIntelligence{rdb: rdb}
}

func (ci *CollectiveIntelligence) ProcessUserReports(ctx context.Context, routeID string, reports []UserReport) (*BusConsensus, error) {
	if len(reports) < MinVotesForConsensus {
		return nil, fmt.Errorf("insufficient reports: need %d, got %d", MinVotesForConsensus, len(reports))
	}

	medianLat := ci.calculateMedian(ci.extractLats(reports))
	medianLng := ci.calculateMedian(ci.extractLngs(reports))
	medianSpeed := ci.calculateMedian(ci.extractSpeeds(reports))
	medianHeading := ci.calculateMedian(ci.extractHeadings(reports))

	filteredReports := ci.filterOutliers(reports, medianLat, medianLng)
	if len(filteredReports) < MinVotesForConsensus {
		return nil, fmt.Errorf("too many outliers: need %d valid reports", MinVotesForConsensus)
	}

	confidence := ci.calculateConfidence(filteredReports)

	consensus := &BusConsensus{
		RouteID:      routeID,
		Latitude:     medianLat,
		Longitude:    medianLng,
		Speed:        medianSpeed,
		Heading:      medianHeading,
		Contributors: len(filteredReports),
		Confidence:   confidence,
		Timestamp:    time.Now(),
	}

	if ci.rdb != nil {
		ci.cacheConsensus(ctx, consensus)
	}

	logger.Info("CI", "Consensus calculated | Route: %s | Contributors: %d | Confidence: %.2f",
		routeID, consensus.Contributors, consensus.Confidence)

	return consensus, nil
}

func (ci *CollectiveIntelligence) filterOutliers(reports []UserReport, medianLat, medianLng float64) []UserReport {
	var valid []UserReport
	for _, r := range reports {
		distance := ci.haversine(medianLat, medianLng, r.Latitude, r.Longitude)
		if distance <= MaxOutlierDistanceM {
			valid = append(valid, r)
		}
	}
	return valid
}

func (ci *CollectiveIntelligence) calculateMedian(nums []float64) float64 {
	sort.Float64s(nums)
	n := len(nums)
	if n == 0 {
		return 0
	}
	if n%2 == 0 {
		return (nums[n/2-1] + nums[n/2]) / 2
	}
	return nums[n/2]
}

func (ci *CollectiveIntelligence) extractLats(reports []UserReport) []float64 {
	lats := make([]float64, len(reports))
	for i, r := range reports {
		lats[i] = r.Latitude
	}
	return lats
}

func (ci *CollectiveIntelligence) extractLngs(reports []UserReport) []float64 {
	lngs := make([]float64, len(reports))
	for i, r := range reports {
		lngs[i] = r.Longitude
	}
	return lngs
}

func (ci *CollectiveIntelligence) extractSpeeds(reports []UserReport) []float64 {
	speeds := make([]float64, len(reports))
	for i, r := range reports {
		speeds[i] = r.Speed
	}
	return speeds
}

func (ci *CollectiveIntelligence) extractHeadings(reports []UserReport) []float64 {
	headings := make([]float64, len(reports))
	for i, r := range reports {
		headings[i] = r.Heading
	}
	return headings
}

func (ci *CollectiveIntelligence) calculateConfidence(reports []UserReport) float64 {
	if len(reports) == 0 {
		return 0
	}

	base := float64(len(reports)) / 10.0
	if base > 1.0 {
		base = 1.0
	}

	now := time.Now()
	recencyBonus := 0.0
	for _, r := range reports {
		age := now.Sub(r.Timestamp).Minutes()
		if age < 1 {
			recencyBonus += 0.1
		} else if age < 2 {
			recencyBonus += 0.05
		}
	}

	confidence := base + recencyBonus
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

func (ci *CollectiveIntelligence) haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371000

	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	Δφ := (lat2 - lat1) * math.Pi / 180
	Δλ := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) + math.Cos(φ1)*math.Cos(φ2)*math.Sin(Δλ/2)*math.Sin(Δλ/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func (ci *CollectiveIntelligence) cacheConsensus(ctx context.Context, consensus *BusConsensus) {
	key := fmt.Sprintf("ci:consensus:%s", consensus.RouteID)

	err := ci.rdb.Set(ctx, key, consensus, ConsensusTTL).Err()
	if err != nil {
		logger.Warn("CI", "Failed to cache consensus: %v", err)
	}
}

func (ci *CollectiveIntelligence) GetCachedConsensus(ctx context.Context, routeID string) (*BusConsensus, error) {
	key := fmt.Sprintf("ci:consensus:%s", routeID)

	data, err := ci.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var consensus BusConsensus
	err = json.Unmarshal([]byte(data), &consensus)
	if err != nil {
		return nil, err
	}

	return &consensus, nil
}
