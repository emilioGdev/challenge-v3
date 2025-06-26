package handlers

import (
	"bytes"
	"challenge-v3/models"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockNATSJetStream struct {
	nats.JetStreamContext
	mock.Mock
}

func (m *MockNATSJetStream) Publish(subj string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error) {
	args := m.Called(subj, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*nats.PubAck), args.Error(1)
}

func float64Ptr(f float64) *float64 { return &f }

func TestHandleGyroscope_Async(t *testing.T) {
	mockJS := new(MockNATSJetStream)
	api := NewAPI(nil, nil, mockJS)

	testData := models.GyroscopeData{
		DeviceID:  "gyro-test-async",
		X:         float64Ptr(1),
		Y:         float64Ptr(2),
		Z:         float64Ptr(3),
		Timestamp: time.Now(),
	}
	payloadBytes, err := json.Marshal(testData)
	require.NoError(t, err)
	mockJS.On("Publish", "telemetry.gyroscope", payloadBytes).Return(&nats.PubAck{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/telemetry/gyroscope", bytes.NewBuffer(payloadBytes))
	rr := httptest.NewRecorder()
	api.HandleGyroscope(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	mockJS.AssertExpectations(t)
}

func TestHandleGPS_Async(t *testing.T) {
	mockJS := new(MockNATSJetStream)
	api := NewAPI(nil, nil, mockJS)

	testData := models.GPSData{
		DeviceID:  "gps-test-async",
		Latitude:  float64Ptr(10),
		Longitude: float64Ptr(20),
		Timestamp: time.Now(),
	}
	payloadBytes, err := json.Marshal(testData)
	require.NoError(t, err)
	mockJS.On("Publish", "telemetry.gps", payloadBytes).Return(&nats.PubAck{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/telemetry/gps", bytes.NewBuffer(payloadBytes))
	rr := httptest.NewRecorder()
	api.HandleGPS(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	mockJS.AssertExpectations(t)
}

func TestHandlePhoto_Async(t *testing.T) {

	t.Run("sucesso - publica mensagem na fila", func(t *testing.T) {
		mockJS := new(MockNATSJetStream)
		api := NewAPI(nil, nil, mockJS)

		testData := models.PhotoData{
			DeviceID:   "photo-test-async",
			Photo:      "aW1hZ2VtLWRhdGE=",
			Timestamp:  time.Now(),
			Recognized: false,
		}
		payloadBytes, err := json.Marshal(testData)
		require.NoError(t, err)

		mockJS.On("Publish", "telemetry.photo", payloadBytes).Return(&nats.PubAck{}, nil)

		req := httptest.NewRequest(http.MethodPost, "/telemetry/photo", bytes.NewBuffer(payloadBytes))
		rr := httptest.NewRecorder()
		api.HandlePhoto(rr, req)

		assert.Equal(t, http.StatusAccepted, rr.Code)
		mockJS.AssertExpectations(t)
	})

	t.Run("falha - dados de validação inválidos", func(t *testing.T) {
		mockJS := new(MockNATSJetStream)
		api := NewAPI(nil, nil, mockJS)

		testData := models.PhotoData{DeviceID: "photo-test-invalid"}
		payloadBytes, err := json.Marshal(testData)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/telemetry/photo", bytes.NewBuffer(payloadBytes))
		rr := httptest.NewRecorder()
		api.HandlePhoto(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockJS.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
	})
}
