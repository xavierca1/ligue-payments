package worker

import (
	"context"
	"database/sql"
	"log"
	"time"
)


type PixExpirationWorker struct {
	db               *sql.DB
	expirationWindow time.Duration
	tickInterval     time.Duration
}


func NewPixExpirationWorker(db *sql.DB) *PixExpirationWorker {
	return &PixExpirationWorker{
		db:               db,
		expirationWindow: 30 * time.Minute, // PIX expira em 30 min
		tickInterval:     1 * time.Minute,  // Roda a cada 1 min
	}
}


func (w *PixExpirationWorker) Start(ctx context.Context) {
	log.Println("🕒 PIX Expiration Worker iniciado (30min window)")

	ticker := time.NewTicker(w.tickInterval)
	defer ticker.Stop()


	w.expireOldPix(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("⚠️ PIX Expiration Worker encerrado")
			return
		case <-ticker.C:
			w.expireOldPix(ctx)
		}
	}
}


func (w *PixExpirationWorker) expireOldPix(ctx context.Context) {
	query := `
		UPDATE subscriptions
		SET 
			status = 'EXPIRED',
			updated_at = NOW()
		WHERE 
			status = 'WAITING_PAYMENT'
			AND payment_method = 'PIX'
			AND created_at < NOW() - INTERVAL '30 minutes'
		RETURNING id, customer_id, created_at
	`

	rows, err := w.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("❌ Erro ao buscar PIX expirados: %v", err)
		return
	}
	defer rows.Close()

	expiredCount := 0
	for rows.Next() {
		var subID, customerID string
		var createdAt time.Time

		if err := rows.Scan(&subID, &customerID, &createdAt); err != nil {
			log.Printf("⚠️ Erro ao escanear PIX expirado: %v", err)
			continue
		}

		elapsed := time.Since(createdAt)
		log.Printf("⏱️ PIX expirado: subscription=%s customer=%s elapsed=%s",
			subID, customerID, elapsed.Round(time.Minute))
		expiredCount++
	}

	if expiredCount > 0 {
		log.Printf("✅ %d PIX(s) marcados como EXPIRED", expiredCount)
	}
}
