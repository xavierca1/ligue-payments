package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"

	"github.com/xavierca1/ligue-payments/internal/infra/database"
	"github.com/xavierca1/ligue-payments/internal/usecase"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Aviso: Arquivo .env não encontrado, usando variáveis do sistema")
	}

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("ERRO: A variável DATABASE_URL é obrigatória!")
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("Erro ao abrir conexão: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Erro fatal: Não foi possível conectar ao banco: %v", err)
	}
	fmt.Println("✅ Conectado ao Supabase com sucesso!")

	repo := database.NewCustomerRepository(db)

	uc := usecase.NewCreateCustomerUseCase(repo)

	webHandler := NewWebHandler(uc)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "db": "connected"})
	})

	http.HandleFunc("/customers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método não permitido (use POST)", http.StatusMethodNotAllowed)
			return
		}
		webHandler.HandleCreateCustomer(w, r)
	})

	fmt.Println("🔥 Servidor rodando na porta 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
