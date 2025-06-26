package services

import (
	"challenge-v3/ierr"
	"challenge-v3/models"
	"challenge-v3/storage"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/patrickmn/go-cache"
)

type PhotoAnalyzer interface {
	AnalyzeAndSavePhoto(data *models.PhotoData) (bool, error)
}
type RekognitionClient interface {
	SearchFacesByImage(ctx context.Context, params *rekognition.SearchFacesByImageInput, optFns ...func(*rekognition.Options)) (*rekognition.SearchFacesByImageOutput, error)
	IndexFaces(ctx context.Context, params *rekognition.IndexFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.IndexFacesOutput, error)
}
type PhotoAnalyzerService struct {
	rekognitionClient RekognitionClient
	collectionID      string
	cache             *cache.Cache
	db                storage.Storage
}

func NewPhotoAnalyzerService(rekClient RekognitionClient, collID string, db storage.Storage) *PhotoAnalyzerService {
	return &PhotoAnalyzerService{
		rekognitionClient: rekClient,
		collectionID:      collID,
		cache:             cache.New(5*time.Minute, 10*time.Minute),
		db:                db,
	}
}

func (s *PhotoAnalyzerService) AnalyzeAndSavePhoto(data *models.PhotoData) (bool, error) {
	if err := data.Validate(); err != nil {
		return false, ierr.NewValidationError("dados da foto inválidos: %w", err)
	}
	slog.Info("Iniciando análise da foto")

	imageBytes, err := base64.StdEncoding.DecodeString(data.Photo)
	if err != nil {
		slog.Error("Falha ao decodificar imagem base64", "error", err)
		return false, fmt.Errorf("imagem base64 inválida")
	}

	cacheKey := fmt.Sprintf("%x", sha256.Sum256(imageBytes))
	slog.Debug("Chave de cache gerada para a imagem", "cache_key", cacheKey)

	if recognized, found := s.cache.Get(cacheKey); found {
		slog.Info("Imagem encontrada no cache", "cache_hit", true, "recognized", recognized)
		data.Recognized = recognized.(bool)
		if err := s.db.SavePhoto(data); err != nil {
			return false, err
		}
		return data.Recognized, nil
	}

	slog.Info("Imagem não encontrada no cache, prosseguindo para análise no Rekognition", "cache_hit", false)

	var recognized bool
	searchResult, err := s.rekognitionClient.SearchFacesByImage(context.TODO(), &rekognition.SearchFacesByImageInput{
		CollectionId:       aws.String(s.collectionID),
		Image:              &types.Image{Bytes: imageBytes},
		MaxFaces:           aws.Int32(1),
		FaceMatchThreshold: aws.Float32(90.0),
	})
	if err != nil {
		slog.Error("Falha ao buscar face no Rekognition", "error", err)
		return false, fmt.Errorf("erro ao analisar a imagem")
	}

	if len(searchResult.FaceMatches) > 0 {
		recognized = true
		match := searchResult.FaceMatches[0]
		slog.Info("Rosto reconhecido na coleção",
			"similarity", *match.Similarity,
			"face_id", *match.Face.FaceId)
	} else {
		recognized = false
		slog.Warn("Rosto não reconhecido, tentando indexar novo rosto")
		indexResult, err := s.rekognitionClient.IndexFaces(context.TODO(), &rekognition.IndexFacesInput{
			CollectionId:        aws.String(s.collectionID),
			Image:               &types.Image{Bytes: imageBytes},
			MaxFaces:            aws.Int32(1),
			DetectionAttributes: []types.Attribute{types.AttributeDefault},
		})
		if err != nil || len(indexResult.FaceRecords) == 0 {
			slog.Error("Falha ao indexar novo rosto", "error", err)
		} else {
			slog.Info("Novo rosto indexado com sucesso",
				"face_id", *indexResult.FaceRecords[0].Face.FaceId)
		}
	}

	if recognized {
		slog.Debug("Salvando resultado no cache", "cache_key", cacheKey, "recognized", true)
		s.cache.Set(cacheKey, true, cache.DefaultExpiration)
	}

	data.Recognized = recognized
	if err := s.db.SavePhoto(data); err != nil {
		return false, err
	}

	slog.Info("Análise e salvamento da foto concluídos", "recognized", recognized)
	return recognized, nil
}
