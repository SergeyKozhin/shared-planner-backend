package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/SergeyKozhin/shared-planner-backend/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

type GoogleInfo struct {
	Name        string
	Email       string
	Picture     string
	PhoneNumber string
}

type clientSecrets map[string]creds

type creds struct {
	ClientId                string   `json:"client_id"`
	ProjectId               string   `json:"project_id"`
	AuthUri                 string   `json:"auth_uri"`
	TokenUri                string   `json:"token_uri"`
	AuthProviderX509CertUrl string   `json:"auth_provider_x509_cert_url"`
	ClientSecret            string   `json:"client_secret"`
	RedirectUris            []string `json:"redirect_uris"`
}

func (p *Parser) GetInfoGoogle(ctx context.Context, authCode string) (*GoogleInfo, error) {
	file, err := os.Open(config.ClientSecretPath())
	if err != nil {
		return nil, fmt.Errorf("can't open client secret: %w", err)
	}
	defer file.Close()

	cs := make(clientSecrets)
	if err := json.NewDecoder(file).Decode(&cs); err != nil {
		return nil, fmt.Errorf("can't parse secrets: %w", err)
	}

	secret := cs[config.ClientType()]
	conf := oauth2.Config{
		ClientID:     secret.ClientId,
		ClientSecret: secret.ClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  config.RedirectURL(),
		Scopes: []string{
			people.UserinfoEmailScope,
			people.UserinfoProfileScope,
			people.UserPhonenumbersReadScope,
		},
	}

	token, err := conf.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("code exchange: %w", err)
	}

	peopleService, err := people.NewService(ctx,
		option.WithScopes(
			people.UserinfoEmailScope,
			people.UserinfoProfileScope,
			people.UserPhonenumbersReadScope,
		),
		option.WithTokenSource(conf.TokenSource(ctx, token)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to People API: %w", err)
	}

	resp, err := peopleService.People.
		Get("people/me").
		PersonFields("names,emailAddresses,photos,phoneNumbers").
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to make request for user info: %w", err)
	}

	if resp.HTTPStatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: code: %d", resp.HTTPStatusCode)
	}

	info := &GoogleInfo{}

	for _, n := range resp.Names {
		if n.Metadata.Primary {
			info.Name = n.DisplayName
			break
		}
	}

	for _, e := range resp.EmailAddresses {
		if e.Metadata.Primary {
			info.Email = e.Value
			break
		}
	}

	for _, p := range resp.Photos {
		if p.Metadata.Primary {
			info.Picture = p.Url
			break
		}
	}

	for _, p := range resp.PhoneNumbers {
		if p.Metadata.Primary {
			info.PhoneNumber = p.Value
			break
		}
	}

	return info, nil
}
