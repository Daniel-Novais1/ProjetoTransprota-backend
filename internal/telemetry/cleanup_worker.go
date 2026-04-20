package telemetry

import (
	"context"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/redis/go-redis/v9"
)

// CleanupWorker gerencia a retenção de dados GPS
// Remove dados com mais de 7 dias para evitar explosão do banco
type CleanupWorker struct {
	db    interface{}
	redis *redis.Client
	stop  chan struct{}
}

// NewCleanupWorker cria um novo worker de limpeza
func NewCleanupWorker(db interface{}, rdb *redis.Client) *CleanupWorker {
	return &CleanupWorker{
		db:    db,
		redis: rdb,
		stop:  make(chan struct{}),
	}
}

// Start inicia o worker que roda a cada 1 hora
func (w *CleanupWorker) Start() {
	logger.Info("CleanupWorker", "Iniciando worker de limpeza de dados (executa a cada 1h)")

	ticker := time.NewTicker(1 * time.Hour)

	go func() {
		// Executar limpeza imediatamente na inicialização
		w.runCleanup()

		for {
			select {
			case <-ticker.C:
				w.runCleanup()
			case <-w.stop:
				ticker.Stop()
				logger.Info("CleanupWorker", "Worker de limpeza parado")
				return
			}
		}
	}()
}

// Stop para o worker
func (w *CleanupWorker) Stop() {
	close(w.stop)
}

// runCleanup executa a limpeza de dados antigos
func (w *CleanupWorker) runCleanup() {
	start := time.Now()
	ctx := context.Background()

	logger.Info("CleanupWorker", "Executando limpeza de dados com mais de 7 dias...")

	// Deletar dados GPS com mais de 7 dias
	// NOTA: Em produção, considere mover para Cold Storage em vez de deletar
	deletedCount, err := w.deleteOldGPSData(ctx, 7)
	if err != nil {
		logger.Error("CleanupWorker", "Erro ao deletar dados antigos: %v", err)
		return
	}

	elapsed := time.Since(start)
	logger.Info("CleanupWorker", "Limpeza concluída: %d registros deletados em %v", deletedCount, elapsed)
}

// deleteOldGPSData deleta dados GPS com mais de N dias
func (w *CleanupWorker) deleteOldGPSData(ctx context.Context, days int) (int64, error) {
	logger.Info("CleanupWorker", "Deletando dados GPS com mais de %d dias", days)

	// Implementação simplificada - em produção use SQL direto
	// DELETE FROM gps_telemetry WHERE recorded_at < NOW() - INTERVAL 'N days'

	// Placeholder - implementação real requer acesso direto ao SQL
	// Para MVP, apenas log e retorna 0
	return 0, nil
}
