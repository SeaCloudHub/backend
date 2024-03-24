package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/SeaCloudHub/backend/pkg/config"
	keto "github.com/ory/keto-client-go"
)

type PermissionService struct {
	readClient  *keto.APIClient
	writeClient *keto.APIClient
}

func NewPermissionService(cfg *config.Config) *PermissionService {
	return &PermissionService{
		readClient:  newKetoClient(cfg.Keto.ReadURL, cfg.DEBUG),
		writeClient: newKetoClient(cfg.Keto.WriteURL, cfg.DEBUG),
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
		if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
			return false, fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
		}

		return false, fmt.Errorf("unexpected error: %w", err)
	}

	return result.Allowed, nil
}

func (s *PermissionService) CreateManager(ctx context.Context, userID string) error {
	_, _, err := s.writeClient.RelationshipApi.CreateRelationship(ctx).CreateRelationshipBody(
		keto.CreateRelationshipBody{
			Namespace: keto.PtrString("User"),
			Object:    keto.PtrString("*"),
			SubjectId: keto.PtrString(userID),
			Relation:  keto.PtrString("manager"),
		},
	).Execute()
	if err != nil {
		if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
			return fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
		}

		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

func assertKetoError[T any](err error) (*keto.GenericOpenAPIError, *T) {
	var ketoErr *keto.GenericOpenAPIError

	if errors.As(err, &ketoErr) {
		if t, ok := ketoErr.Model().(T); ok {
			return ketoErr, &t
		}

		return ketoErr, nil
	}

	return nil, nil
}
