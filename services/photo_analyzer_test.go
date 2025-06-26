package services

import (
	"challenge-v3/ierr"
	"challenge-v3/models"
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRekognitionClient struct{ mock.Mock }

func (m *MockRekognitionClient) SearchFacesByImage(ctx context.Context, params *rekognition.SearchFacesByImageInput, optFns ...func(*rekognition.Options)) (*rekognition.SearchFacesByImageOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*rekognition.SearchFacesByImageOutput), args.Error(1)
}
func (m *MockRekognitionClient) IndexFaces(ctx context.Context, params *rekognition.IndexFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.IndexFacesOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*rekognition.IndexFacesOutput), args.Error(1)
}

type MockStorage struct{ mock.Mock }

func (m *MockStorage) SavePhoto(data *models.PhotoData) error         { return m.Called(data).Error(0) }
func (m *MockStorage) SaveGyroscope(data *models.GyroscopeData) error { return m.Called(data).Error(0) }
func (m *MockStorage) SaveGPS(data *models.GPSData) error             { return m.Called(data).Error(0) }

func validTestPhoto() models.PhotoData {
	return models.PhotoData{
		DeviceID:  "test-device",
		Photo:     "dGVzdA==",
		Timestamp: time.Now(),
	}
}

func TestPhotoAnalyzer_FaceRecognized(t *testing.T) {
	mockRek := new(MockRekognitionClient)
	mockDB := new(MockStorage)
	photoAnalyzer := NewPhotoAnalyzerService(mockRek, "test-collection", mockDB)
	testPhoto := validTestPhoto()

	faceID, similarity := "test-face-id", float32(99.9)
	searchOutput := &rekognition.SearchFacesByImageOutput{
		FaceMatches: []types.FaceMatch{{Face: &types.Face{FaceId: &faceID}, Similarity: &similarity}},
	}
	mockRek.On("SearchFacesByImage", mock.Anything, mock.Anything).Return(searchOutput, nil)
	mockDB.On("SavePhoto", mock.MatchedBy(func(p *models.PhotoData) bool { return p.Recognized })).Return(nil)

	recognized, err := photoAnalyzer.AnalyzeAndSavePhoto(&testPhoto)

	assert.NoError(t, err)
	assert.True(t, recognized)
	mockRek.AssertExpectations(t)
	mockDB.AssertExpectations(t)
}

func TestPhotoAnalyzer_FaceNotRecognized_AndIndexed(t *testing.T) {
	mockRek := new(MockRekognitionClient)
	mockDB := new(MockStorage)
	photoAnalyzer := NewPhotoAnalyzerService(mockRek, "test-collection", mockDB)
	testPhoto := validTestPhoto()
	mockRek.On("SearchFacesByImage", mock.Anything, mock.Anything).Return(&rekognition.SearchFacesByImageOutput{}, nil)

	faceID := "new-face-id"
	indexOutput := &rekognition.IndexFacesOutput{FaceRecords: []types.FaceRecord{{Face: &types.Face{FaceId: &faceID}}}}
	mockRek.On("IndexFaces", mock.Anything, mock.Anything).Return(indexOutput, nil)
	mockDB.On("SavePhoto", mock.MatchedBy(func(p *models.PhotoData) bool { return !p.Recognized })).Return(nil)

	recognized, err := photoAnalyzer.AnalyzeAndSavePhoto(&testPhoto)

	assert.NoError(t, err)
	assert.False(t, recognized)
	mockRek.AssertExpectations(t)
	mockDB.AssertExpectations(t)
}

func TestPhotoAnalyzer_ValidationFail(t *testing.T) {
	mockRek := new(MockRekognitionClient)
	mockDB := new(MockStorage)
	photoAnalyzer := NewPhotoAnalyzerService(mockRek, "test-collection", mockDB)

	testPhoto := models.PhotoData{Photo: "dGVzdA=="}

	recognized, err := photoAnalyzer.AnalyzeAndSavePhoto(&testPhoto)

	assert.False(t, recognized)
	assert.Error(t, err)

	var validationErr *ierr.ValidationError
	assert.ErrorAs(t, err, &validationErr, "O erro deveria ser do tipo ValidationError")
}
