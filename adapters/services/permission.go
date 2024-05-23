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

func (s *PermissionService) GetDirectoryUsers(ctx context.Context, fileID string) ([]permission.FileUser, error) {
	var (
		fileUsers []permission.FileUser
		first     = true
		cursor    string
	)

	for first || len(cursor) > 0 {
		result, _, err := s.readClient.RelationshipApi.GetRelationships(ctx).PageSize(100).PageToken(cursor).
			Namespace("Directory").Object(fileID).Execute()
		if err != nil {
			if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
				return nil, fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
			}

			return nil, fmt.Errorf("unexpected error: %w", err)
		}

		for _, relationship := range result.RelationTuples {
			if role, ok := permission.RelationshipRoleMap[relationship.Relation]; ok {
				fileUsers = append(fileUsers, permission.FileUser{
					UserID: *relationship.SubjectId,
					Role:   role,
				})
			}
		}

		cursor = *result.NextPageToken
		first = false
	}

	return fileUsers, nil
}

func (s *PermissionService) GetSharedPermissions(ctx context.Context, userID string, namespace string, relation string) ([]string, error) {
	var (
		sharedIDs []string
		first     = true
		cursor    string
	)

	for first || len(cursor) > 0 {
		result, _, err := s.readClient.RelationshipApi.GetRelationships(ctx).PageSize(100).PageToken(cursor).
			Namespace(namespace).SubjectId(userID).Relation(relation).Execute()
		if err != nil {
			if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
				return nil, fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
			}

			return nil, fmt.Errorf("unexpected error: %w", err)
		}

		for _, relationship := range result.RelationTuples {
			sharedIDs = append(sharedIDs, relationship.Object)
		}

		cursor = *result.NextPageToken
		first = false
	}

	return sharedIDs, nil
}

func (s *PermissionService) GetFileUserRoles(ctx context.Context,
	userID string, fileID string, isDir bool) ([]string, error) {
	var (
		permissions []string
	)

	namespace := "File"
	if isDir {
		namespace = "Directory"
	}

	result, _, err := s.readClient.RelationshipApi.GetRelationships(ctx).PageSize(100).
		Namespace(namespace).Object(fileID).SubjectId(userID).Execute()
	if err != nil {
		if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
			return nil, fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
		}

		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	for _, relationship := range result.RelationTuples {
		if role, ok := permission.RelationshipRoleMap[relationship.Relation]; ok {
			permissions = append(permissions, role)
		}
	}

	return permissions, nil
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

func (s *PermissionService) CanDeleteDirectory(ctx context.Context, userID string, fileID string) (bool, error) {
	result, _, err := s.readClient.PermissionApi.CheckPermission(ctx).
		Namespace("Directory").Object(fileID).SubjectId(userID).Relation("delete").
		Execute()
	if err != nil {
		if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
			return false, fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
		}

		return false, fmt.Errorf("unexpected error: %w", err)
	}

	return result.Allowed, nil
}

func (s *PermissionService) IsDirectoryOwner(ctx context.Context, userID string, fileID string) (bool, error) {
	result, _, err := s.readClient.PermissionApi.CheckPermission(ctx).
		Namespace("Directory").Object(fileID).SubjectId(userID).Relation("owners").
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

func (s *PermissionService) UpdateDirectoryParent(ctx context.Context, fileID string, parentID string, oldParentID string) error {
	_, err := s.writeClient.RelationshipApi.PatchRelationships(ctx).
		RelationshipPatch([]keto.RelationshipPatch{
			{
				Action: keto.PtrString("delete"),
				RelationTuple: &keto.Relationship{
					Namespace: "Directory",
					Object:    fileID,
					SubjectId: keto.PtrString(oldParentID),
					Relation:  "parents",
				},
			},
			{
				Action: keto.PtrString("insert"),
				RelationTuple: &keto.Relationship{
					Namespace: "Directory",
					Object:    fileID,
					SubjectId: keto.PtrString(parentID),
					Relation:  "parents",
				},
			},
		}).Execute()
	if err != nil {
		if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
			return fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
		}

		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

func (s *PermissionService) DeleteDirectoryPermissions(ctx context.Context, fileID string) error {
	_, err := s.writeClient.RelationshipApi.DeleteRelationships(ctx).Namespace("Directory").Object(fileID).Execute()
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

func (s *PermissionService) CanDeleteFile(ctx context.Context, userID string, fileID string) (bool, error) {
	result, _, err := s.readClient.PermissionApi.CheckPermission(ctx).
		Namespace("File").Object(fileID).SubjectId(userID).Relation("delete").
		Execute()
	if err != nil {
		if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
			return false, fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
		}

		return false, fmt.Errorf("unexpected error: %w", err)
	}

	return result.Allowed, nil
}

func (s *PermissionService) IsFileOwner(ctx context.Context, userID string, fileID string) (bool, error) {
	result, _, err := s.readClient.PermissionApi.CheckPermission(ctx).
		Namespace("File").Object(fileID).SubjectId(userID).Relation("owners").
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

func (s *PermissionService) UpdateFileParent(ctx context.Context, fileID string, parentID string, oldParentID string) error {
	_, err := s.writeClient.RelationshipApi.PatchRelationships(ctx).
		RelationshipPatch([]keto.RelationshipPatch{
			{
				Action: keto.PtrString("delete"),
				RelationTuple: &keto.Relationship{
					Namespace: "File",
					Object:    fileID,
					SubjectId: keto.PtrString(oldParentID),
					Relation:  "parents",
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
		}).Execute()
	if err != nil {
		if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
			return fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
		}

		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

func (s *PermissionService) DeleteFilePermissions(ctx context.Context, fileID string) error {
	_, err := s.writeClient.RelationshipApi.DeleteRelationships(ctx).Namespace("File").Object(fileID).Execute()
	if err != nil {
		if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
			return fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
		}

		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

func (s *PermissionService) GetFileUsers(ctx context.Context, fileID string) ([]permission.FileUser, error) {
	var (
		fileUsers []permission.FileUser
		first     = true
		cursor    string
	)

	for first || len(cursor) > 0 {
		result, _, err := s.readClient.RelationshipApi.GetRelationships(ctx).PageSize(100).PageToken(cursor).
			Namespace("File").Object(fileID).Execute()
		if err != nil {
			if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
				return nil, fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
			}

			return nil, fmt.Errorf("unexpected error: %w", err)
		}

		for _, relationship := range result.RelationTuples {
			if role, ok := permission.RelationshipRoleMap[relationship.Relation]; ok {
				fileUsers = append(fileUsers, permission.FileUser{
					UserID: *relationship.SubjectId,
					Role:   role,
				})
			}
		}

		cursor = *result.NextPageToken
		first = false
	}

	return fileUsers, nil
}

func (s *PermissionService) DeleteUserPermissions(ctx context.Context, userID string) error {
	_, err := s.writeClient.RelationshipApi.DeleteRelationships(ctx).Namespace("File").SubjectId(userID).Execute()
	if err != nil {
		if _, genericErr := assertKetoError[keto.ErrorGeneric](err); genericErr != nil {
			return fmt.Errorf("unexpected error: %s", genericErr.Error.GetReason())
		}

		return fmt.Errorf("unexpected error: %w", err)
	}

	_, err = s.writeClient.RelationshipApi.DeleteRelationships(ctx).Namespace("Directory").SubjectId(userID).Execute()
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
