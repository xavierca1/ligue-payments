package database

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Driver do Postgres
)

// NewDBConnection abre a conexão e testa o Ping
func NewDBConnection(connString string) (*sql.DB, error) {
	// 1. Abre a conexão (mas não conecta de verdade ainda, só valida a string)
	db, err := sql.Open("pgx", connString)
	if err != nil {
		return nil, err
	}

	// 2. Configura o Pool (Essencial para produção)
	db.SetMaxOpenConns(10) // Máximo de conexões abertas
	db.SetMaxIdleConns(5)  // Máximo de conexões paradas esperando uso
	db.SetConnMaxLifetime(5 * time.Minute)

	// 3. O Ping: A prova de fogo
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err // Retorna erro se o Supabase não responder
	}

	return db, nil
}
