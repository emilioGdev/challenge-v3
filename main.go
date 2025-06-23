package main

import (
	"challenge-v3/handlers"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/telemetry/gyroscope", handlers.HandleGyroscope)
	http.HandleFunc("/telemetry/gps", handlers.HandleGPS)
	http.HandleFunc("/telemetry/photo", handlers.HandlePhoto)

	log.Println("Servidor iniciado e escutando na porta :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
