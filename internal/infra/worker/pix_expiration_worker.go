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

	// w.expireOldPix(ctx)

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
	expiredStatus, err := w.resolveExpiredStatusLabel(ctx)
	if err != nil {
		log.Printf("❌ Erro ao resolver enum sub_status para expiração PIX: %v", err)
		return
	}
	if expiredStatus == "" {
		log.Printf("⚠️ Nenhum status de expiração válido encontrado no enum sub_status (EXPIRED/CANCELED/CANCELLED)")
		return
	}

	query := `
		UPDATE subscriptions
		SET 
			status = $1::sub_status,
			updated_at = NOW()
		WHERE 
			status = 'PENDING'
			AND payment_method = 'PIX'
			AND created_at < NOW() - INTERVAL '30 minutes'
		RETURNING id, customer_id, created_at
	`

	rows, err := w.db.QueryContext(ctx, query, expiredStatus)
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
		log.Printf("✅ %d PIX(s) marcados como %s", expiredCount, expiredStatus)
	}
}

func (w *PixExpirationWorker) resolveExpiredStatusLabel(ctx context.Context) (string, error) {
	rows, err := w.db.QueryContext(ctx, `SELECT unnest(enum_range(NULL::sub_status)::text[])`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	labels := map[string]bool{}
	for rows.Next() {
		var label string
		if err := rows.Scan(&label); err != nil {
			return "", err
		}
		labels[label] = true
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	if labels["EXPIRED"] {
		return "EXPIRED", nil
	}
	if labels["CANCELED"] {
		return "CANCELED", nil
	}
	if labels["CANCELLED"] {
		return "CANCELLED", nil
	}

	return "", nil
}
