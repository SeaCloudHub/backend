package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/SeaCloudHub/backend/pkg/config"
	keto "github.com/ory/keto-client-go"
)

type PermissionService struct {
	readClient  *keto.APIClient
	writeClient *keto.APIClient
}

func NewPermissionService(cfg *config.Config) *PermissionService {
	var debug bool
	if strings.ToLower(cfg.AppEnv) == "local" {
		debug = true
	}

	return &PermissionService{
		readClient:  newKetoClient(cfg.Keto.ReadURL, debug),
		writeClient: newKetoClient(cfg.Keto.WriteURL, debug),
	}
}

func newKetoClient(url string, debug bool) *keto.APIClient {
	configuration := keto.NewConfiguration()
	configuration.Servers = keto.ServerConfigurations{{URL: url}}
	configuration.Debug = debug

	return keto.NewAPIClient(configuration)
}

func (s *PermissionService) IsManager(ctx context.Context, userID string) (bool, error) {
	result, _, err := s.readClient.PermissionApi.CheckPermission(ctx).
		Namespace("User").
		Object("*").
		SubjectId(userID).
		Relation("manager").
		Execute()
	if err != nil {
		if _, ok := assetKetoError[keto.ErrorGeneric](err); ok {
			return false, errors.New("unexpected error")
		}

		return false, fmt.Errorf("unexpected error: %w", err)
	}

	return result.Allowed, nil
}

func assetKetoError[T any](err error) (*keto.GenericOpenAPIError, bool) {
	var ketoErr *keto.GenericOpenAPIError

	if errors.As(err, &ketoErr) {
		if _, ok := ketoErr.Model().(T); ok {
			return ketoErr, true
		}

		return ketoErr, false
	}

	return nil, false
}
