package main

import (
	"challenge-v3/ierr"
	"challenge-v3/messaging"
	"challenge-v3/models"
	"challenge-v3/services"
	"challenge-v3/storage"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
)

type Worker struct {
	photoAnalyzer services.PhotoAnalyzer
}

func (w *Worker) handlePhotoMsg(msg *nats.Msg) {
	log.Println("Nova mensagem de foto recebida.")
	var data models.PhotoData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		log.Printf("ERRO: Falha ao decodificar mensagem JSON: %v. Mensagem terminada.", err)
		msg.Term()
		return
	}

	_, err := w.photoAnalyzer.AnalyzeAndSavePhoto(&data)
	if err != nil {
		var validationErr *ierr.ValidationError
		if errors.As(err, &validationErr) {
			log.Printf("ERRO de validação ao processar foto: %v. Mensagem terminada.", err)
			msg.Term()
		} else {
			log.Printf("ERRO: Falha ao processar foto: %v. A mensagem será reenviada.", err)
			msg.Nak()
		}
		return
	}

	log.Printf("Mensagem para device %s processada com sucesso.", data.DeviceID)
	msg.Ack()
}

func main() {
	log.Println("Iniciando o Worker...")
	godotenv.Load()

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
	db, err := storage.NewPostgresStorage(connStr)
	if err != nil {
		log.Fatalf("ERRO: DB: %v", err)
	}
	awsRegion := os.Getenv("AWS_REGION")
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(awsRegion))
	if err != nil {
		log.Fatalf("ERRO: Falha ao carregar config da AWS: %v", err)
	}
	rekognitionClient := rekognition.NewFromConfig(cfg)
	collectionID := os.Getenv("REKOGNITION_COLLECTION_ID")
	photoAnalyzer := services.NewPhotoAnalyzerService(rekognitionClient, collectionID, db)
	natsURL := os.Getenv("NATS_URL")
	nc, err := messaging.ConnectNATS(natsURL)
	if err != nil {
		log.Fatalf("ERRO: Falha ao conectar ao NATS: %v", err)
	}
	defer nc.Close()
	js, err := messaging.SetupJetStream(nc)
	if err != nil {
		log.Fatalf("ERRO: Falha ao configurar o JetStream: %v", err)
	}

	worker := &Worker{
		photoAnalyzer: photoAnalyzer,
	}

	js.Subscribe("telemetry.photo", worker.handlePhotoMsg, nats.Durable("PHOTO_WORKER"), nats.AckWait(30*time.Second))

	log.Println("Worker está no ar, esperando por mensagens...")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Println("Desligando o Worker...")
}
