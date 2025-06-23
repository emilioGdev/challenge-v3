package main

import (
	"challenge-v3/handlers"
	"challenge-v3/storage"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv" //
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Aviso: Não foi possível encontrar o arquivo .env, usando variáveis de ambiente do sistema.")
	}

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)
	db, err := storage.NewPostgresStorage(connStr)
	if err != nil {
		log.Fatalf("ERRO: Não foi possível conectar ao banco de dados: %v", err)
	}

	if err := db.InitTables(); err != nil {
		log.Fatalf("ERRO: Não foi possível inicializar as tabelas: %v", err)
	}

	api := handlers.NewAPI(db)

	http.HandleFunc("/telemetry/gyroscope", api.HandleGyroscope)
	http.HandleFunc("/telemetry/gps", api.HandleGPS)
	http.HandleFunc("/telemetry/photo", api.HandlePhoto)

	log.Println("Servidor iniciado e escutando na porta :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
