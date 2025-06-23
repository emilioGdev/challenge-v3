package handlers

import (
	"challenge-v3/models"
	"encoding/json"
	"log"
	"net/http"
)

func SendJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(models.ErrorResponse{Message: message})
}

func SendJSONSuccess(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}

func HandleGyroscope(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Método não permitido. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var data models.GyroscopeData
	err := decoder.Decode(&data)
	if err != nil {
		log.Printf("ERRO: Falha ao decodificar JSON para giroscópio: %v", err)
		SendJSONError(w, "Corpo da requisição inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := data.Validate(); err != nil {
		SendJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	SendJSONSuccess(w, "Dados de giroscópio recebidos com sucesso!", http.StatusOK)
}

func HandleGPS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Método não permitido. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var data models.GPSData
	err := decoder.Decode(&data)
	if err != nil {
		log.Printf("ERRO: Falha ao decodificar JSON para GPS: %v", err)
		SendJSONError(w, "Corpo da requisição inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := data.Validate(); err != nil {
		SendJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	SendJSONSuccess(w, "Dados de GPS recebidos com sucesso!", http.StatusOK)
}

func HandlePhoto(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Método não permitido. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var data models.PhotoData
	err := decoder.Decode(&data)
	if err != nil {
		log.Printf("ERRO: Falha ao decodificar JSON para foto: %v", err)
		SendJSONError(w, "Corpo da requisição inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := data.Validate(); err != nil {
		SendJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	SendJSONSuccess(w, "Foto recebida com sucesso!", http.StatusOK)
}
