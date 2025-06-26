package main

import (
	"challenge-v3/handlers"
	"challenge-v3/messaging"
	"challenge-v3/services"
	"challenge-v3/storage"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
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

	awsRegion := os.Getenv("AWS_REGION")
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(awsRegion))
	if err != nil {
		log.Fatalf("ERRO: Falha ao carregar configuração da AWS: %v", err)
	}

	natsURL := os.Getenv("NATS_URL") // Adicione NATS_URL=nats://nats:4222 ao seu .env
	nc, err := messaging.ConnectNATS(natsURL)
	if err != nil {
		log.Fatalf("ERRO: Falha ao conectar ao NATS: %v", err)
	}
	defer nc.Close()

	js, err := messaging.SetupJetStream(nc)
	if err != nil {
		log.Fatalf("ERRO: Falha ao configurar o JetStream: %v", err)
	}

	rekognitionClient := rekognition.NewFromConfig(cfg)
	collectionID := os.Getenv("REKOGNITION_COLLECTION_ID")

	_, err = rekognitionClient.CreateCollection(context.TODO(), &rekognition.CreateCollectionInput{CollectionId: &collectionID})

	var resourceExistsErr *types.ResourceAlreadyExistsException
	if err != nil {
		if errors.As(err, &resourceExistsErr) {
			log.Printf("AVISO: Coleção '%s' já existe. Continuando.\n", collectionID)
		} else {
			log.Fatalf("ERRO: Falha ao criar/verificar coleção no Rekognition: %v", err)
		}
	} else {
		log.Printf("Coleção '%s' criada com sucesso.\n", collectionID)
	}

	photoAnalyzer := services.NewPhotoAnalyzerService(rekognitionClient, collectionID, db)

	api := handlers.NewAPI(db, photoAnalyzer, js)

	http.HandleFunc("/telemetry/gyroscope", api.HandleGyroscope)
	http.HandleFunc("/telemetry/gps", api.HandleGPS)
	http.HandleFunc("/telemetry/photo", api.HandlePhoto)

	log.Println("Servidor iniciado e escutando na porta :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

}
