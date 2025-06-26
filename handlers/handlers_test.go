package handlers

import (
	"bytes"
	"challenge-v3/models"
	"challenge-v3/services"
	"challenge-v3/storage"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"os"

	"github.com/joho/godotenv"
)

func setupTestAPI(t *testing.T) (*API, *sql.DB) {
	err := godotenv.Load("../.env")
	if err != nil {
		t.Log("Aviso: Arquivo .env não encontrado, usando variáveis do sistema.")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	dbStorage, err := storage.NewPostgresStorage(connStr)
	require.NoError(t, err, "Falha ao conectar ao storage do postgres")

	err = dbStorage.InitTables()
	require.NoError(t, err, "Falha ao inicializar tabelas")

	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err, "Falha ao abrir conexão sql crua")

	_, err = db.Exec("TRUNCATE TABLE gyroscope, gps, photo RESTART IDENTITY")
	require.NoError(t, err, "Falha ao limpar tabelas")

	awsRegion := os.Getenv("AWS_REGION")
	require.NotEmpty(t, awsRegion, "AWS_REGION não pode ser vazio para os testes")

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(awsRegion))
	require.NoError(t, err, "Falha ao carregar configuração da AWS para o teste")

	rekognitionClient := rekognition.NewFromConfig(cfg)
	collectionID := os.Getenv("REKOGNITION_COLLECTION_ID")
	require.NotEmpty(t, collectionID, "REKOGNITION_COLLECTION_ID não pode ser vazio para os testes")

	photoAnalyzer := services.NewPhotoAnalyzerService(rekognitionClient, collectionID, dbStorage)

	return NewAPI(dbStorage, photoAnalyzer), db
}

func TestHandleGyroscope_Integration(t *testing.T) {
	api, db := setupTestAPI(t)
	defer db.Close()

	t.Run("sucesso_salva_no_db", func(t *testing.T) {
		payload := `{"device_id": "gyro-test-ok", "x": 10.1, "y": 20.2, "z": 30.3, "timestamp": "2025-06-23T12:00:00Z"}`
		req := httptest.NewRequest(http.MethodPost, "/telemetry/gyroscope", bytes.NewBufferString(payload))
		rr := httptest.NewRecorder()
		api.HandleGyroscope(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var deviceID string
		var x float64
		err := db.QueryRow("SELECT device_id, x FROM gyroscope WHERE device_id = 'gyro-test-ok'").Scan(&deviceID, &x)
		require.NoError(t, err, "Dado de giroscópio não foi encontrado no banco de dados")
		assert.Equal(t, "gyro-test-ok", deviceID)
		assert.InDelta(t, 10.1, x, 0.001)
	})

	t.Run("falha_campo_faltando", func(t *testing.T) {
		payload := `{"device_id": "gyro-test-fail", "x": 10.1, "y": 20.2, "timestamp": "2025-06-23T12:00:00Z"}` // Falta 'z'
		req := httptest.NewRequest(http.MethodPost, "/telemetry/gyroscope", bytes.NewBufferString(payload))
		rr := httptest.NewRecorder()
		api.HandleGyroscope(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "campo obrigatório ausente: z")
	})
}

func TestHandleGPS_Integration(t *testing.T) {
	api, db := setupTestAPI(t)
	defer db.Close()

	t.Run("sucesso_salva_no_db", func(t *testing.T) {
		payload := `{"device_id": "gps-test-ok", "latitude": -8.05, "longitude": -34.88, "timestamp": "2025-06-23T13:00:00Z"}`
		req := httptest.NewRequest(http.MethodPost, "/telemetry/gps", bytes.NewBufferString(payload))
		rr := httptest.NewRecorder()
		api.HandleGPS(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var deviceID string
		var latitude float64
		err := db.QueryRow("SELECT device_id, latitude FROM gps WHERE device_id = 'gps-test-ok'").Scan(&deviceID, &latitude)
		require.NoError(t, err, "Dado de GPS não foi encontrado no banco de dados")
		assert.Equal(t, "gps-test-ok", deviceID)
		assert.InDelta(t, -8.05, latitude, 0.001)
	})

	t.Run("falha_campo_desconhecido", func(t *testing.T) {
		payload := `{"device_id": "gps-test-fail", "latitude": 1, "longitude": 2, "timestamp": "2025-06-23T13:00:00Z", "extra_field": "some_value"}`
		req := httptest.NewRequest(http.MethodPost, "/telemetry/gps", bytes.NewBufferString(payload))
		rr := httptest.NewRecorder()
		api.HandleGPS(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		var errorResponse models.ErrorResponse
		err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
		require.NoError(t, err, "Falha ao decodificar a resposta de erro JSON")
		assert.Contains(t, errorResponse.Message, "json: unknown field \"extra_field\"")
	})
}

func TestHandlePhoto_Integration(t *testing.T) {
	api, db := setupTestAPI(t)
	defer db.Close()

	t.Run("sucesso_salva_no_db", func(t *testing.T) {
		photoData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="
		payload := fmt.Sprintf(`{"device_id": "photo-test-ok", "photo": "%s", "timestamp": "2025-06-23T14:00:00Z"}`, photoData)
		req := httptest.NewRequest(http.MethodPost, "/telemetry/photo", bytes.NewBufferString(payload))
		rr := httptest.NewRecorder()
		api.HandlePhoto(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var returnedPhoto string
		err := db.QueryRow("SELECT photo FROM photo WHERE device_id = 'photo-test-ok'").Scan(&returnedPhoto)
		require.NoError(t, err, "Dado de Foto não foi encontrado no banco de dados")
		assert.Equal(t, photoData, returnedPhoto)
	})

	t.Run("falha_photo_vazio", func(t *testing.T) {
		payload := `{"device_id": "photo-test-fail", "photo": "", "timestamp": "2025-06-23T14:00:00Z"}`
		req := httptest.NewRequest(http.MethodPost, "/telemetry/photo", bytes.NewBufferString(payload))
		rr := httptest.NewRecorder()
		api.HandlePhoto(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "campo obrigatório ausente: photo")
	})
}
