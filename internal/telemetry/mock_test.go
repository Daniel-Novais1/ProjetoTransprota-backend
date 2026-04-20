package telemetry

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/redis/go-redis/v9"
)

// MockPostgresDB simula conexão PostgreSQL para testes
type MockPostgresDB struct {
	ShouldFail      bool
	ShouldTimeout   bool
	SavedData       []TelemetryPing
	LatestPositions []LatestPosition
}

// MockRedisClient simula cliente Redis para testes
type MockRedisClient struct {
	ShouldFail     bool
	ShouldTimeout  bool
	Cache          map[string]string
	PubSubMessages chan string
	Subscriptions  map[string]bool
}

// NewMockPostgresDB cria mock de PostgreSQL
func NewMockPostgresDB() *MockPostgresDB {
	return &MockPostgresDB{
		SavedData:       make([]TelemetryPing, 0),
		LatestPositions: make([]LatestPosition, 0),
	}
}

// NewMockRedisClient cria mock de Redis
func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		Cache:          make(map[string]string),
		PubSubMessages: make(chan string, 100),
		Subscriptions:  make(map[string]bool),
	}
}

// Ping simula ping do Redis
func (m *MockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	if m.ShouldFail {
		cmd.SetErr(fmt.Errorf("mock redis failure"))
	} else {
		cmd.SetVal("PONG")
	}
	return cmd
}

// Get simula GET do Redis
func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	if m.ShouldFail {
		cmd.SetErr(fmt.Errorf("mock redis failure"))
		return cmd
	}
	if val, ok := m.Cache[key]; ok {
		cmd.SetVal(val)
	} else {
		cmd.SetErr(redis.Nil)
	}
	return cmd
}

// Set simula SET do Redis
func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	if m.ShouldFail {
		cmd.SetErr(fmt.Errorf("mock redis failure"))
		return cmd
	}
	m.Cache[key] = fmt.Sprintf("%v", value)
	cmd.SetVal("OK")
	return cmd
}

// Del simula DEL do Redis
func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	if m.ShouldFail {
		cmd.SetErr(fmt.Errorf("mock redis failure"))
		return cmd
	}
	count := 0
	for _, key := range keys {
		if _, ok := m.Cache[key]; ok {
			delete(m.Cache, key)
			count++
		}
	}
	cmd.SetVal(int64(count))
	return cmd
}

// SAdd simula SADD do Redis
func (m *MockRedisClient) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	if m.ShouldFail {
		cmd.SetErr(fmt.Errorf("mock redis failure"))
		return cmd
	}
	cmd.SetVal(int64(len(members)))
	return cmd
}

// SCard simula SCARD do Redis
func (m *MockRedisClient) SCard(ctx context.Context, key string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	if m.ShouldFail {
		cmd.SetErr(fmt.Errorf("mock redis failure"))
		return cmd
	}
	cmd.SetVal(0)
	return cmd
}

// Publish simula PUBLISH do Redis
func (m *MockRedisClient) Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	if m.ShouldFail {
		cmd.SetErr(fmt.Errorf("mock redis failure"))
		return cmd
	}
	select {
	case m.PubSubMessages <- fmt.Sprintf("%v", message):
	default:
		logger.Warn("MockRedis", "PubSub channel full")
	}
	cmd.SetVal(1)
	return cmd
}

// Subscribe simula SUBSCRIBE do Redis
func (m *MockRedisClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	pubsub := &redis.PubSub{}
	m.Subscriptions = make(map[string]bool)
	for _, ch := range channels {
		m.Subscriptions[ch] = true
	}
	return pubsub
}

// Close simula Close do Redis
func (m *MockRedisClient) Close() error {
	if m.ShouldFail {
		return fmt.Errorf("mock redis close failure")
	}
	return nil
}

// Shutdown simula Shutdown do Redis
func (m *MockRedisClient) Shutdown(ctx context.Context) error {
	return m.Close()
}

// TestMockRedisSuccess testa mock Redis com sucesso
func TestMockRedisSuccess(t *testing.T) {
	ctx := context.Background()
	mockRedis := NewMockRedisClient()

	// Test Set
	err := mockRedis.Set(ctx, "test-key", "test-value", time.Minute).Err()
	if err != nil {
		t.Errorf("Set falhou: %v", err)
	}

	// Test Get
	val, err := mockRedis.Get(ctx, "test-key").Result()
	if err != nil {
		t.Errorf("Get falhou: %v", err)
	}
	if val != "test-value" {
		t.Errorf("Valor incorreto: esperado 'test-value', recebeu '%s'", val)
	}

	// Test Publish
	err = mockRedis.Publish(ctx, "test-channel", "test-message").Err()
	if err != nil {
		t.Errorf("Publish falhou: %v", err)
	}

	// Test Ping
	err = mockRedis.Ping(ctx).Err()
	if err != nil {
		t.Errorf("Ping falhou: %v", err)
	}
}

// TestMockRedisFailure testa mock Redis com falha
func TestMockRedisFailure(t *testing.T) {
	ctx := context.Background()
	mockRedis := NewMockRedisClient()
	mockRedis.ShouldFail = true

	// Test Set com falha
	err := mockRedis.Set(ctx, "test-key", "test-value", time.Minute).Err()
	if err == nil {
		t.Error("Set deveria falhar")
	}

	// Test Get com falha
	_, err = mockRedis.Get(ctx, "test-key").Result()
	if err == nil {
		t.Error("Get deveria falhar")
	}

	// Test Publish com falha
	err = mockRedis.Publish(ctx, "test-channel", "test-message").Err()
	if err == nil {
		t.Error("Publish deveria falhar")
	}
}

// TestMockPostgresDB testa mock de PostgreSQL
func TestMockPostgresDB(t *testing.T) {
	mockDB := NewMockPostgresDB()

	// Test inicialização
	if mockDB.SavedData == nil {
		t.Error("SavedData não inicializado")
	}
	if mockDB.LatestPositions == nil {
		t.Error("LatestPositions não inicializado")
	}

	// Test adicionar dados
	mockDB.SavedData = append(mockDB.SavedData, TelemetryPing{
		DeviceID:  "test-1",
		Latitude:  -16.6869,
		Longitude: -49.2648,
	})

	if len(mockDB.SavedData) != 1 {
		t.Errorf("Esperado 1 item, recebeu %d", len(mockDB.SavedData))
	}
}

// TestRepositoryWithMock testa Repository com mock Redis
func TestRepositoryWithMock(t *testing.T) {
	t.Skip("Skipping - requires real PostgreSQL connection")
}
