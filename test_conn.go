package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

func main() {
	fmt.Println("=== Testando Conexões do TranspRota Backend ===\n")

	// Configurações
	dbHost := "127.0.0.1"
	dbPort := "5432"
	dbUser := "admin"
	dbPassword := "password123"
	dbName := "transprota"
	redisAddr := "127.0.0.1:6379"

	// Testar PostgreSQL
	fmt.Println("1. Testando conexão PostgreSQL...")
	fmt.Printf("   Host: %s, Port: %s, Database: %s, User: %s\n", dbHost, dbPort, dbName, dbUser)
	
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable host=%s port=%s",
		dbUser, dbPassword, dbName, dbHost, dbPort)
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("   ❌ ERRO ao abrir conexão PostgreSQL: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("   ❌ ERRO no PING PostgreSQL: %v", err)
	}
	
	fmt.Println("   ✅ PostgreSQL conectado com sucesso!\n")

	// Testar Redis
	fmt.Println("2. Testando conexão Redis...")
	fmt.Printf("   Addr: %s\n", redisAddr)
	
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer rdb.Close()

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("   ❌ ERRO no PING Redis: %v", err)
	}
	
	fmt.Println("   ✅ Redis conectado com sucesso!\n")

	fmt.Println("=== Todas as conexões funcionando! ===")
}
