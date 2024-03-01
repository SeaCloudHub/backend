package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

func (s *IdentityService) Login(ctx context.Context, email string, password string) (string, error) {
	flow, _, err := s.publicClient.FrontendAPI.CreateNativeLoginFlow(ctx).Execute()
	if err != nil {
		return "", err
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
			return "", identity.ErrInvalidCredentials
		}

		return "", fmt.Errorf("unexpected error: %w", err)
	}

	return result.GetSessionToken(), nil
}

func (s *IdentityService) WhoAmI(ctx context.Context, token string) (*identity.Identity, error) {
	session, _, err := s.publicClient.FrontendAPI.ToSession(ctx).XSessionToken(token).Execute()
	if err != nil {
		if _, genericErr := assetKratosError[kratos.ErrorGeneric](err); genericErr != nil {
			return nil, identity.ErrInvalidSession
		}

		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	return &identity.Identity{
		ID: session.Identity.Id,
	}, nil
}

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
				"email": email,
			},
		},
	).Execute()
	if err != nil {
		if _, genericErr := assetKratosError[kratos.ErrorGeneric](err); genericErr != nil {
			return nil, fmt.Errorf("error creating identity: %s", genericErr.Error.GetMessage())
		}

		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	return &identity.Identity{
		ID: id.Id,
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
