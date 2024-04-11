package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/SeaCloudHub/backend/domain/permission"
	"github.com/SeaCloudHub/backend/pkg/config"
	keto "github.com/ory/keto-client-go"
)

type PermissionService struct {
	readClient  *keto.APIClient
	writeClient *keto.APIClient
}

func NewPermissionService(cfg *config.Config) *PermissionService {
	return &PermissionService{
		readClient:  newKetoClient(cfg.Keto.ReadURL, cfg.Debug),
		writeClient: newKetoClient(cfg.Keto.WriteURL, cfg.Debug),
	}
}

func newKetoClient(url string, debug bool) *keto.APIClient {
	configuration := keto.NewConfiguration()
	configuration.Servers = keto.ServerConfigurations{{URL: url}}
	configuration.Debug = debug

	return keto.NewAPIClient(configuration)
}

func (s *PermissionService) IsAdmin(ctx context.Context, userID string) (bool, error) {
	result, _, err := s.readClient.PermissionApi.CheckPermission(ctx).
		Namespace("Group").Object("admins").SubjectId(userID).Relation("members").
		Execute()
	if err != nil {
		if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
			return false, fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
		}

		return false, fmt.Errorf("unexpected error: %w", err)
	}

	return result.Allowed, nil
}

func (s *PermissionService) CreateAdminGroup(ctx context.Context, userID string) error {
	_, _, err := s.writeClient.RelationshipApi.CreateRelationship(ctx).CreateRelationshipBody(
		keto.CreateRelationshipBody{
			Namespace: keto.PtrString("Group"),
			Object:    keto.PtrString("admins"),
			SubjectId: keto.PtrString(userID),
			Relation:  keto.PtrString("members"),
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

func (s *PermissionService) CreatePermission(ctx context.Context, in *permission.CreatePermission) error {
	_, _, err := s.writeClient.RelationshipApi.CreateRelationship(ctx).CreateRelationshipBody(
		keto.CreateRelationshipBody{
			Namespace: keto.PtrString(in.Namespace),
			Object:    keto.PtrString(in.FileID),
			SubjectId: keto.PtrString(in.UserID),
			Relation:  keto.PtrString(in.Relation),
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

func (s *PermissionService) CreateDirectoryPermissions(ctx context.Context, userID string, fileID string, parentID string) error {
	relationshipPatch := []keto.RelationshipPatch{
		{
			Action: keto.PtrString("insert"),
			RelationTuple: &keto.Relationship{
				Namespace: "Directory",
				Object:    fileID,
				SubjectId: keto.PtrString(userID),
				Relation:  "owners",
			},
		},
		{
			Action: keto.PtrString("insert"),
			RelationTuple: &keto.Relationship{
				Namespace: "Directory",
				Object:    fileID,
				SubjectSet: &keto.SubjectSet{
					Namespace: "Group",
					Object:    "admins",
					Relation:  "members",
				},
				Relation: "managers",
			},
		},
	}

	if parentID != "" {
		relationshipPatch = append(relationshipPatch, keto.RelationshipPatch{
			Action: keto.PtrString("insert"),
			RelationTuple: &keto.Relationship{
				Namespace: "Directory",
				Object:    fileID,
				SubjectId: keto.PtrString(parentID),
				Relation:  "parents",
			},
		})
	}

	_, err := s.writeClient.RelationshipApi.PatchRelationships(ctx).
		RelationshipPatch(relationshipPatch).Execute()
	if err != nil {
		if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
			return fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
		}

		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

func (s *PermissionService) CanEditDirectory(ctx context.Context, userID string, fileID string) (bool, error) {
	result, _, err := s.readClient.PermissionApi.CheckPermission(ctx).
		Namespace("Directory").Object(fileID).SubjectId(userID).Relation("edit").
		Execute()
	if err != nil {
		if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
			return false, fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
		}

		return false, fmt.Errorf("unexpected error: %w", err)
	}

	return result.Allowed, nil
}

func (s *PermissionService) CanViewDirectory(ctx context.Context, userID string, fileID string) (bool, error) {
	result, _, err := s.readClient.PermissionApi.CheckPermission(ctx).
		Namespace("Directory").Object(fileID).SubjectId(userID).Relation("view").
		Execute()
	if err != nil {
		if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
			return false, fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
		}

		return false, fmt.Errorf("unexpected error: %w", err)
	}

	return result.Allowed, nil
}

func (s *PermissionService) ClearDirectoryPermissions(ctx context.Context, fileID string, userID string) error {
	_, err := s.writeClient.RelationshipApi.DeleteRelationships(ctx).
		Namespace("Directory").Object(fileID).SubjectId(userID).Execute()
	if err != nil {
		if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
			return fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
		}

		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

func (s *PermissionService) CreateFilePermissions(ctx context.Context, userID string, fileID string, parentID string) error {
	_, err := s.writeClient.RelationshipApi.PatchRelationships(ctx).RelationshipPatch(
		[]keto.RelationshipPatch{
			{
				Action: keto.PtrString("insert"),
				RelationTuple: &keto.Relationship{
					Namespace: "File",
					Object:    fileID,
					SubjectId: keto.PtrString(userID),
					Relation:  "owners",
				},
			},
			{
				Action: keto.PtrString("insert"),
				RelationTuple: &keto.Relationship{
					Namespace: "File",
					Object:    fileID,
					SubjectSet: &keto.SubjectSet{
						Namespace: "Group",
						Object:    "admins",
						Relation:  "members",
					},
					Relation: "managers",
				},
			},
			{
				Action: keto.PtrString("insert"),
				RelationTuple: &keto.Relationship{
					Namespace: "File",
					Object:    fileID,
					SubjectId: keto.PtrString(parentID),
					Relation:  "parents",
				},
			},
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

func (s *PermissionService) CanEditFile(ctx context.Context, userID string, fileID string) (bool, error) {
	result, _, err := s.readClient.PermissionApi.CheckPermission(ctx).
		Namespace("File").Object(fileID).SubjectId(userID).Relation("edit").
		Execute()
	if err != nil {
		if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
			return false, fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
		}

		return false, fmt.Errorf("unexpected error: %w", err)
	}

	return result.Allowed, nil
}

func (s *PermissionService) CanViewFile(ctx context.Context, userID string, fileID string) (bool, error) {
	result, _, err := s.readClient.PermissionApi.CheckPermission(ctx).
		Namespace("File").Object(fileID).SubjectId(userID).Relation("view").
		Execute()
	if err != nil {
		if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
			return false, fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
		}

		return false, fmt.Errorf("unexpected error: %w", err)
	}

	return result.Allowed, nil
}

func (s *PermissionService) ClearFilePermissions(ctx context.Context, fileID string, userID string) error {
	_, err := s.writeClient.RelationshipApi.DeleteRelationships(ctx).
		Namespace("File").Object(fileID).SubjectId(userID).Execute()
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
