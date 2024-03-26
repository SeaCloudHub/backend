package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SeaCloudHub/backend/domain/identity"
	"golang.org/x/crypto/bcrypt"

	"github.com/SeaCloudHub/backend/pkg/config"
	kratos "github.com/ory/kratos-client-go"
	"github.com/ory/x/pagination/keysetpagination"
)

type IdentityService struct {
	publicClient *kratos.APIClient
	adminClient  *kratos.APIClient
}

func NewIdentityService(cfg *config.Config) *IdentityService {
	return &IdentityService{
		publicClient: newKratosClient(cfg.Kratos.PublicURL, cfg.DEBUG),
		adminClient:  newKratosClient(cfg.Kratos.AdminURL, cfg.DEBUG),
	}
}

func newKratosClient(url string, debug bool) *kratos.APIClient {
	configuration := kratos.NewConfiguration()
	configuration.Servers = kratos.ServerConfigurations{{URL: url}}
	configuration.Debug = debug

	return kratos.NewAPIClient(configuration)
}

func (s *IdentityService) Login(ctx context.Context, email string, password string) (*identity.Session, error) {
	flow, _, err := s.publicClient.FrontendAPI.CreateNativeLoginFlow(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	result, _, err := s.publicClient.FrontendAPI.
		UpdateLoginFlow(ctx).Flow(flow.Id).
		UpdateLoginFlowBody(kratos.UpdateLoginFlowBody{
			UpdateLoginFlowWithPasswordMethod: &kratos.UpdateLoginFlowWithPasswordMethod{
				Identifier: email,
				Method:     "password",
				Password:   password,
			},
		}).Execute()
	if err != nil {
		if _, loginFlow := assertKratosError[kratos.LoginFlow](err); loginFlow != nil {
			return nil, identity.ErrInvalidCredentials
		}

		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	id, err := mapIdentity(result.Session.Identity)
	if err != nil {
		return nil, fmt.Errorf("map identity: %w", err)
	}

	return &identity.Session{
		ID:        result.Session.Id,
		Token:     result.SessionToken,
		ExpiresAt: result.Session.ExpiresAt,
		Identity:  id,
	}, nil
}

func (s *IdentityService) WhoAmI(ctx context.Context, token string) (*identity.Identity, error) {
	session, _, err := s.publicClient.FrontendAPI.ToSession(ctx).XSessionToken(token).Execute()
	if err != nil {
		if _, genericErr := assertKratosError[kratos.ErrorGeneric](err); genericErr != nil {
			return nil, identity.ErrInvalidSession
		}

		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	id, err := mapIdentity(session.Identity)
	if err != nil {
		return nil, err
	}

	id.Session = &identity.Session{
		ID:        session.Id,
		Token:     &token,
		ExpiresAt: session.ExpiresAt,
	}

	return id, nil
}

func (s *IdentityService) ChangePassword(ctx context.Context, id *identity.Identity, oldPassword string, newPassword string) error {
	// Get current identity
	iden, _, err := s.adminClient.IdentityAPI.GetIdentity(ctx, id.ID).IncludeCredential([]string{"password"}).Execute()
	if err != nil {
		if _, genericErr := assertKratosError[kratos.ErrorGeneric](err); genericErr != nil {
			return fmt.Errorf("error getting identity: %s", genericErr.Error.GetReason())
		}

		return fmt.Errorf("unexpected error: %w", err)
	}

	// Check if the old password is correct
	if bcrypt.CompareHashAndPassword([]byte((*iden.Credentials)["password"].Config["hashed_password"].(string)), []byte(oldPassword)) != nil {
		return identity.ErrIncorrectPassword
	}

	// Change the password
	flow, _, err := s.publicClient.FrontendAPI.CreateNativeSettingsFlow(ctx).
		XSessionToken(*id.Session.Token).Execute()
	if err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}

	_, _, err = s.publicClient.FrontendAPI.UpdateSettingsFlow(ctx).
		Flow(flow.Id).XSessionToken(*id.Session.Token).UpdateSettingsFlowBody(
		kratos.UpdateSettingsFlowBody{
			UpdateSettingsFlowWithPasswordMethod: &kratos.UpdateSettingsFlowWithPasswordMethod{
				Method:   "password",
				Password: newPassword,
			},
		},
	).Execute()
	if err != nil {
		if _, settingsFlow := assertKratosError[kratos.SettingsFlow](err); settingsFlow != nil {
			return identity.ErrInvalidPassword
		}

		if _, genericErr := assertKratosError[kratos.ErrorGeneric](err); genericErr != nil &&
			genericErr.Error.GetId() == "session_refresh_required" {
			return identity.ErrSessionTooOld
		}

		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

func (s *IdentityService) SetPasswordChangedAt(ctx context.Context, id *identity.Identity) error {
	// Change the profile to update the password_changed_at
	flow, _, err := s.publicClient.FrontendAPI.CreateNativeSettingsFlow(ctx).
		XSessionToken(*id.Session.Token).Execute()
	if err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}

	_, _, err = s.publicClient.FrontendAPI.UpdateSettingsFlow(ctx).
		Flow(flow.Id).XSessionToken(*id.Session.Token).UpdateSettingsFlowBody(
		kratos.UpdateSettingsFlowBody{
			UpdateSettingsFlowWithProfileMethod: &kratos.UpdateSettingsFlowWithProfileMethod{
				Method: "profile",
				Traits: map[string]interface{}{
					"email":               id.Email,
					"first_name":          id.FirstName,
					"last_name":           id.LastName,
					"avatar_url":          id.AvatarURL,
					"password_changed_at": time.Now().Format(time.RFC3339),
				},
			},
		},
	).Execute()
	if err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

func (s *IdentityService) IsEmailExists(ctx context.Context, email string) (bool, error) {
	identities, _, err := s.adminClient.IdentityAPI.ListIdentities(ctx).CredentialsIdentifier(email).Execute()
	if err != nil {
		if _, genericErr := assertKratosError[kratos.ErrorGeneric](err); genericErr != nil {
			return false, fmt.Errorf("error checking email: %s", genericErr.Error.GetReason())
		}

		return false, fmt.Errorf("unexpected error: %w", err)
	}

	if len(identities) > 0 {
		return true, nil
	}

	return false, nil
}

// Admin APIs
func (s *IdentityService) CreateIdentity(ctx context.Context, in identity.SimpleIdentity) (*identity.Identity, error) {
	id, _, err := s.adminClient.IdentityAPI.CreateIdentity(ctx).CreateIdentityBody(
		kratos.CreateIdentityBody{
			Credentials: &kratos.IdentityWithCredentials{
				Password: &kratos.IdentityWithCredentialsPassword{
					Config: &kratos.IdentityWithCredentialsPasswordConfig{
						Password: kratos.PtrString(in.Password),
					},
				},
			},
			Traits: map[string]interface{}{
				"email":               in.Email,
				"first_name":          in.FirstName,
				"last_name":           in.LastName,
				"avatar_url":          in.AvatarURL,
				"password_changed_at": nil,
			},
		},
	).Execute()
	if err != nil {
		if _, genericErr := assertKratosError[kratos.ErrorGeneric](err); genericErr != nil {
			return nil, fmt.Errorf("error creating identity: %s", genericErr.Error.GetReason())
		}

		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	return mapIdentity(id)
}

func (s *IdentityService) CreateMultipleIdentities(ctx context.Context, simpleIdentities []identity.SimpleIdentity) ([]*identity.Identity, error) {
	var identitiesPatch []kratos.IdentityPatch
	for _, simpleIdentity := range simpleIdentities {
		identitiesPatch = append(identitiesPatch, kratos.IdentityPatch{
			Create: &kratos.CreateIdentityBody{
				Credentials: &kratos.IdentityWithCredentials{
					Password: &kratos.IdentityWithCredentialsPassword{
						Config: &kratos.IdentityWithCredentialsPasswordConfig{
							Password: kratos.PtrString(simpleIdentity.Password),
						},
					},
				},
				Traits: map[string]interface{}{
					"email":               simpleIdentity.Email,
					"password_changed_at": nil,
				},
			},
		})
	}

	res, _, err := s.adminClient.IdentityAPI.BatchPatchIdentities(ctx).PatchIdentitiesBody(
		kratos.PatchIdentitiesBody{
			Identities: identitiesPatch}).Execute()
	if err != nil {
		if _, genericErr := assertKratosError[kratos.ErrorGeneric](err); genericErr != nil {
			return nil, fmt.Errorf("error creating identities: %s", genericErr.Error.Message)
		}

		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	return mapIdentityFromPatchRes(res)
}

func (s *IdentityService) ListIdentities(ctx context.Context, pageToken string, pageSize int64) ([]identity.Identity, string, error) {
	req := s.adminClient.IdentityAPI.ListIdentities(ctx).PageSize(pageSize)
	if len(pageToken) > 0 {
		req = req.PageToken(pageToken)
	}

	identities, resp, err := req.Execute()
	if err != nil {
		if _, genericErr := assertKratosError[kratos.ErrorGeneric](err); genericErr != nil {
			return nil, "", fmt.Errorf("error listing identities: %s", genericErr.Error.GetReason())
		}

		return nil, "", fmt.Errorf("unexpected error: %w", err)
	}

	var result []identity.Identity
	for _, id := range identities {
		i, err := mapIdentity(&id)
		if err != nil {
			return nil, "", err
		}

		result = append(result, *i)
	}

	pagination := keysetpagination.ParseHeader(resp)

	return result, pagination.NextToken, nil

}

func mapIdentity(id *kratos.Identity) (*identity.Identity, error) {
	traits, ok := id.GetTraits().(map[string]interface{})
	if !ok {
		return nil, errors.New("get traits")
	}

	email, ok := traits["email"].(string)
	if !ok {
		return nil, errors.New("get email")
	}

	firstName, _ := traits["first_name"].(string)
	lastName, _ := traits["last_name"].(string)
	avatarURL, _ := traits["avatar_url"].(string)

	var passwordChangedAt *time.Time
	if pca, ok := traits["password_changed_at"]; ok && pca != nil && len(pca.(string)) > 0 {
		t, err := time.Parse(time.RFC3339, pca.(string))
		if err != nil {
			return nil, fmt.Errorf("parse password_changed_at: %w", err)
		}

		passwordChangedAt = &t
	}

	return &identity.Identity{
		ID:                id.Id,
		Email:             email,
		FirstName:         firstName,
		LastName:          lastName,
		AvatarURL:         avatarURL,
		PasswordChangedAt: passwordChangedAt,
	}, nil
}

func mapIdentityFromPatchRes(res *kratos.BatchPatchIdentitiesResponse) ([]*identity.Identity, error) {
	var identities []*identity.Identity
	for _, id := range res.Identities {
		i := &identity.Identity{
			ID: *id.Identity,
		}

		identities = append(identities, i)
	}

	return identities, nil
}

func assertKratosError[T any](err error) (*kratos.GenericOpenAPIError, *T) {
	var kratosErr *kratos.GenericOpenAPIError

	if errors.As(err, &kratosErr) {
		if t, ok := kratosErr.Model().(T); ok {
			return kratosErr, &t
		}

		return kratosErr, nil
	}

	return nil, nil
}
