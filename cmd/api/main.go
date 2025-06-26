package main

import (
	_ "challenge-v3/docs"
	"challenge-v3/handlers"
	"challenge-v3/messaging"
	"challenge-v3/metrics"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title           API de Telemetria de Frota
// @version         1.0
// @description     Esta é a API para ingestão de dados de telemetria do Desafio Cloud.
// @host      localhost:8080
// @BasePath  /
func main() {
	godotenv.Load()
	log.Println("Iniciando a API...")

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

	api := handlers.NewAPI(nil, nil, js)

	router := http.NewServeMux()
	router.Handle("/telemetry/gyroscope", metrics.PrometheusMiddleware(http.HandlerFunc(api.HandleGyroscope)))
	router.Handle("/telemetry/gps", metrics.PrometheusMiddleware(http.HandlerFunc(api.HandleGPS)))
	router.Handle("/telemetry/photo", metrics.PrometheusMiddleware(http.HandlerFunc(api.HandlePhoto)))
	router.HandleFunc("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	metricsRouter := http.NewServeMux()
	metricsRouter.Handle("/metrics", metrics.MetricsHandler())
	go func() {
		log.Println("Servidor de Métricas da API iniciado na porta :8081")
		if err := http.ListenAndServe(":8081", metricsRouter); err != nil {
			log.Fatalf("Servidor de Métricas da API falhou: %v", err)
		}
	}()

	log.Println("Servidor da API iniciado na porta :8080. Documentação disponível em http://localhost:8080/swagger/index.html")
	log.Fatal(http.ListenAndServe(":8080", router))
}
