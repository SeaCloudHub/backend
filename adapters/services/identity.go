package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SeaCloudHub/backend/domain/identity"

	"github.com/SeaCloudHub/backend/pkg/config"
	kratos "github.com/ory/kratos-client-go"
)

type IdentityService struct {
	publicClient *kratos.APIClient
	adminClient  *kratos.APIClient
}

func NewIdentityService(cfg *config.Config) *IdentityService {
	var debug bool
	if strings.ToLower(cfg.AppEnv) == "local" {
		debug = true
	}

	return &IdentityService{
		publicClient: newKratosClient(cfg.Kratos.PublicURL, debug),
		adminClient:  newKratosClient(cfg.Kratos.AdminURL, debug),
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
		if _, loginFlow := assetKratosError[kratos.LoginFlow](err); loginFlow != nil {
			return nil, identity.ErrInvalidCredentials
		}

		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	return &identity.Session{
		ID:        result.Session.Id,
		Token:     result.SessionToken,
		ExpiresAt: result.Session.ExpiresAt,
	}, nil
}

func (s *IdentityService) WhoAmI(ctx context.Context, token string) (*identity.Identity, error) {
	session, _, err := s.publicClient.FrontendAPI.ToSession(ctx).XSessionToken(token).Execute()
	if err != nil {
		if _, genericErr := assetKratosError[kratos.ErrorGeneric](err); genericErr != nil {
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
	// Login to check if the old password is correct
	session, err := s.Login(ctx, id.Email, oldPassword)
	if err != nil {
		return err
	}

	// Remove the session that was created by the login
	if _, err := s.publicClient.FrontendAPI.DisableMySession(ctx, session.ID).
		XSessionToken(*id.Session.Token).Execute(); err != nil {
		return fmt.Errorf("unexpected error: %w", err)
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
		if _, settingsFlow := assetKratosError[kratos.SettingsFlow](err); settingsFlow != nil {
			return identity.ErrInvalidCredentials
		}

		return fmt.Errorf("unexpected error: %w", err)
	}

	return nil
}

func (s *IdentityService) SyncPasswordChangedAt(ctx context.Context, id *identity.Identity) error {
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

// Admin APIs
func (s *IdentityService) CreateIdentity(ctx context.Context, email string, password string) (*identity.Identity, error) {
	id, _, err := s.adminClient.IdentityAPI.CreateIdentity(ctx).CreateIdentityBody(
		kratos.CreateIdentityBody{
			Credentials: &kratos.IdentityWithCredentials{
				Password: &kratos.IdentityWithCredentialsPassword{
					Config: &kratos.IdentityWithCredentialsPasswordConfig{
						Password: kratos.PtrString(password),
					},
				},
			},
			Traits: map[string]interface{}{
				"email":               email,
				"password_changed_at": nil,
			},
		},
	).Execute()
	if err != nil {
		if _, genericErr := assetKratosError[kratos.ErrorGeneric](err); genericErr != nil {
			return nil, fmt.Errorf("error creating identity: %s", genericErr.Error.GetMessage())
		}

		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	return mapIdentity(id)
}

func mapIdentity(id *kratos.Identity) (*identity.Identity, error) {
	traits, ok := id.GetTraits().(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot get traits")
	}

	email, ok := traits["email"].(string)
	if !ok {
		return nil, fmt.Errorf("cannot get email")
	}

	var passwordChangedAt *time.Time
	if pca, ok := traits["password_changed_at"]; ok && pca != nil && len(pca.(string)) > 0 {
		t, err := time.Parse(time.RFC3339, pca.(string))
		if err != nil {
			return nil, fmt.Errorf("cannot parse password_changed_at: %w", err)
		}

		passwordChangedAt = &t
	}

	return &identity.Identity{
		ID:                id.Id,
		Email:             email,
		PasswordChangedAt: passwordChangedAt,
	}, nil
}

func assetKratosError[T any](err error) (*kratos.GenericOpenAPIError, *T) {
	var kratosErr *kratos.GenericOpenAPIError

	if errors.As(err, &kratosErr) {
		if t, ok := kratosErr.Model().(T); ok {
			return kratosErr, &t
		}

		return kratosErr, nil
	}

	return nil, nil
}
