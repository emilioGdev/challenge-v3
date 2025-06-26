package handlers

import (
	"challenge-v3/ierr"
	"challenge-v3/models"
	"challenge-v3/services"
	"challenge-v3/storage"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type API struct {
	db            storage.Storage
	photoAnalyzer *services.PhotoAnalyzerService
}

func NewAPI(db storage.Storage, pa *services.PhotoAnalyzerService) *API {
	return &API{
		db:            db,
		photoAnalyzer: pa,
	}
}

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

func (a *API) HandleGyroscope(w http.ResponseWriter, r *http.Request) {
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

	if err := a.db.SaveGyroscope(&data); err != nil {
		log.Printf("ERRO: Falha ao salvar dados de giroscópio: %v", err)
		SendJSONError(w, "Erro interno ao salvar os dados", http.StatusInternalServerError)
		return
	}

	SendJSONSuccess(w, "Dados de giroscópio recebidos e salvos com sucesso!", http.StatusOK)

}

func (a *API) HandleGPS(w http.ResponseWriter, r *http.Request) {
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

	if err := a.db.SaveGPS(&data); err != nil {
		log.Printf("ERRO: Falha ao salvar dados de GPS: %v", err)
		SendJSONError(w, "Erro interno ao salvar os dados", http.StatusInternalServerError)
		return
	}
	SendJSONSuccess(w, "Dados de GPS recebidos e salvos com sucesso!", http.StatusOK)
}

func (a *API) HandlePhoto(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONError(w, "Método não permitido. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var data models.PhotoData
	if err := decoder.Decode(&data); err != nil {
		SendJSONError(w, "Corpo da requisição inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	recognized, err := a.photoAnalyzer.AnalyzeAndSavePhoto(&data)
	if err != nil {
		var validationErr *ierr.ValidationError
		if errors.As(err, &validationErr) {
			SendJSONError(w, validationErr.Error(), http.StatusBadRequest)
		} else {
			log.Printf("ERRO INTERNO: %v", err)
			SendJSONError(w, "erro interno ao processar a foto", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Foto recebida, analisada e salva com sucesso!",
		"recognized": recognized,
	})
}
