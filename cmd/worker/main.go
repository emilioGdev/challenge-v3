package main

import (
	"challenge-v3/ierr"
	"challenge-v3/messaging"
	"challenge-v3/metrics"
	"challenge-v3/models"
	"challenge-v3/services"
	"challenge-v3/storage"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
)

type Worker struct {
	db            storage.Storage
	photoAnalyzer services.PhotoAnalyzer
}

func (w *Worker) handleGyroscopeMsg(msg *nats.Msg) {
	subject := "telemetry.gyroscope"
	log.Printf("Nova mensagem recebida em '%s'.\n", subject)
	var data models.GyroscopeData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		log.Printf("ERRO: Falha ao decodificar mensagem de giroscópio: %v. Mensagem terminada.", err)
		msg.Term()
		metrics.NatsMessagesProcessed.WithLabelValues(subject, "terminated").Inc()
		return
	}
	if err := w.db.SaveGyroscope(&data); err != nil {
		log.Printf("ERRO: Falha ao salvar dados de giroscópio: %v. A mensagem será reenviada.", err)
		msg.Nak()
		metrics.NatsMessagesProcessed.WithLabelValues(subject, "failed").Inc()
		return
	}
	log.Printf("Mensagem de giroscópio para device %s processada com sucesso.", data.DeviceID)
	msg.Ack()
	metrics.NatsMessagesProcessed.WithLabelValues(subject, "success").Inc()
}

func (w *Worker) handleGpsMsg(msg *nats.Msg) {
	subject := "telemetry.gps"
	log.Printf("Nova mensagem recebida em '%s'.\n", subject)
	var data models.GPSData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		log.Printf("ERRO: Falha ao decodificar mensagem de GPS: %v. Mensagem terminada.", err)
		msg.Term()
		metrics.NatsMessagesProcessed.WithLabelValues(subject, "terminated").Inc()
		return
	}
	if err := w.db.SaveGPS(&data); err != nil {
		log.Printf("ERRO: Falha ao salvar dados de GPS: %v. A mensagem será reenviada.", err)
		msg.Nak()
		metrics.NatsMessagesProcessed.WithLabelValues(subject, "failed").Inc()
		return
	}
	log.Printf("Mensagem de GPS para device %s processada com sucesso.", data.DeviceID)
	msg.Ack()
	metrics.NatsMessagesProcessed.WithLabelValues(subject, "success").Inc()
}

func (w *Worker) handlePhotoMsg(msg *nats.Msg) {
	subject := "telemetry.photo"
	log.Printf("Nova mensagem recebida em '%s'.\n", subject)
	var data models.PhotoData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		log.Printf("ERRO: Falha ao decodificar mensagem JSON: %v. Mensagem terminada.", err)
		msg.Term()
		metrics.NatsMessagesProcessed.WithLabelValues(subject, "terminated").Inc()
		return
	}
	_, err := w.photoAnalyzer.AnalyzeAndSavePhoto(&data)
	if err != nil {
		var validationErr *ierr.ValidationError
		if errors.As(err, &validationErr) {
			log.Printf("ERRO de validação ao processar foto: %v. Mensagem terminada.", err)
			msg.Term()
			metrics.NatsMessagesProcessed.WithLabelValues(subject, "terminated").Inc()
		} else {
			log.Printf("ERRO: Falha ao processar foto: %v. A mensagem será reenviada.", err)
			msg.Nak()
			metrics.NatsMessagesProcessed.WithLabelValues(subject, "failed").Inc()
		}
		return
	}
	log.Printf("Mensagem de foto para device %s processada com sucesso.", data.DeviceID)
	msg.Ack()
	metrics.NatsMessagesProcessed.WithLabelValues(subject, "success").Inc()
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
	_, err = rekognitionClient.CreateCollection(context.TODO(), &rekognition.CreateCollectionInput{CollectionId: &collectionID})
	var resourceExistsErr *types.ResourceAlreadyExistsException
	if err != nil && !errors.As(err, &resourceExistsErr) {
		log.Fatalf("ERRO: Falha ao criar/verificar coleção no Rekognition: %v", err)
	}
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

	metricsRouter := http.NewServeMux()
	metricsRouter.Handle("/metrics", metrics.MetricsHandler())
	go func() {
		log.Println("Servidor de Métricas do Worker iniciado na porta :8082")
		if err := http.ListenAndServe(":8082", metricsRouter); err != nil {
			log.Fatalf("Servidor de Métricas do Worker falhou: %v", err)
		}
	}()

	worker := &Worker{
		db:            db,
		photoAnalyzer: photoAnalyzer,
	}

	ackWait30s := nats.AckWait(30 * time.Second)

	_, err = js.Subscribe("telemetry.gyroscope", worker.handleGyroscopeMsg, nats.Durable("GYROSCOPE_WORKER"))
	if err != nil {
		log.Fatalf("Falha ao se inscrever no tópico de giroscópio: %v", err)
	}

	_, err = js.Subscribe("telemetry.gps", worker.handleGpsMsg, nats.Durable("GPS_WORKER"))
	if err != nil {
		log.Fatalf("Falha ao se inscrever no tópico de GPS: %v", err)
	}

	_, err = js.Subscribe("telemetry.photo", worker.handlePhotoMsg, nats.Durable("PHOTO_WORKER"), ackWait30s)
	if err != nil {
		log.Fatalf("Falha ao se inscrever no tópico de fotos: %v", err)
	}
	log.Println("Worker está no ar, esperando por todas as mensagens de telemetria...")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Println("Desligando o Worker...")
}
