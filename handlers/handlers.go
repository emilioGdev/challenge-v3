package handlers

import (
	"challenge-v3/models"
	"challenge-v3/services"
	"challenge-v3/storage"
	"encoding/json"
	"log"
	"net/http"

	"github.com/nats-io/nats.go"
)

type API struct {
	db            storage.Storage
	photoAnalyzer services.PhotoAnalyzer
	natsJS        nats.JetStreamContext
}

func NewAPI(db storage.Storage, pa services.PhotoAnalyzer, js nats.JetStreamContext) *API {
	return &API{
		db:            db,
		photoAnalyzer: pa,
		natsJS:        js,
	}
}

func SendJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(models.ErrorResponse{Message: message})
}

// --- Handlers Assíncronos (Versão Nível 5) ---

func (a *API) HandleGyroscope(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Método não permitido. Use POST.", http.StatusMethodNotAllowed)
		return
	}
	var data models.GyroscopeData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		SendJSONError(w, "Corpo da requisição inválido", http.StatusBadRequest)
		return
	}
	if err := data.Validate(); err != nil {
		SendJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	msgData, err := json.Marshal(data)
	if err != nil {
		SendJSONError(w, "Erro interno ao preparar a mensagem", http.StatusInternalServerError)
		return
	}

	_, err = a.natsJS.Publish("telemetry.gyroscope", msgData)
	if err != nil {
		log.Printf("ERRO: Falha ao publicar mensagem no NATS: %v", err)
		SendJSONError(w, "Erro interno ao enviar dados para processamento", http.StatusInternalServerError)
		return
	}

	log.Printf("Mensagem para device %s publicada em 'telemetry.gyroscope'", data.DeviceID)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"message": "Dados de giroscópio recebidos e enfileirados."})
}

func (a *API) HandleGPS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Método não permitido. Use POST.", http.StatusMethodNotAllowed)
		return
	}
	var data models.GPSData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		SendJSONError(w, "Corpo da requisição inválido", http.StatusBadRequest)
		return
	}
	if err := data.Validate(); err != nil {
		SendJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	msgData, err := json.Marshal(data)
	if err != nil {
		SendJSONError(w, "Erro interno ao preparar a mensagem", http.StatusInternalServerError)
		return
	}

	_, err = a.natsJS.Publish("telemetry.gps", msgData)
	if err != nil {
		log.Printf("ERRO: Falha ao publicar mensagem no NATS: %v", err)
		SendJSONError(w, "Erro interno ao enviar dados para processamento", http.StatusInternalServerError)
		return
	}

	log.Printf("Mensagem para device %s publicada em 'telemetry.gps'", data.DeviceID)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"message": "Dados de GPS recebidos e enfileirados."})
}

func (a *API) HandlePhoto(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Método não permitido. Use POST.", http.StatusMethodNotAllowed)
		return
	}
	var data models.PhotoData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		SendJSONError(w, "Corpo da requisição inválido", http.StatusBadRequest)
		return
	}

	if err := data.Validate(); err != nil {
		SendJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	msgData, err := json.Marshal(data)
	if err != nil {
		SendJSONError(w, "Erro interno ao preparar a mensagem", http.StatusInternalServerError)
		return
	}

	_, err = a.natsJS.Publish("telemetry.photo", msgData)
	if err != nil {
		log.Printf("ERRO: Falha ao publicar mensagem no NATS: %v", err)
		SendJSONError(w, "Erro interno ao enviar dados para processamento", http.StatusInternalServerError)
		return
	}

	log.Printf("Mensagem para device %s publicada em 'telemetry.photo'", data.DeviceID)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"message": "Dados da foto recebidos e enfileirados para processamento."})
}
