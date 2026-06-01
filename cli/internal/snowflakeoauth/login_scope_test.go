package snowflakeoauth

import (
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

func TestAuthCodeURLOmitsScopeWhenUnset(t *testing.T) {
	issuer, err := IssuerForEnv("prod")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &oauth2.Config{
		ClientID:    "test-client",
		RedirectURL: DefaultRedirectURI,
		Endpoint: oauth2.Endpoint{
			AuthURL: issuer.AuthorizeURL,
		},
	}
	authURL := cfg.AuthCodeURL("state")
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	if q := u.Query().Get("scope"); q != "" {
		t.Fatalf("scope query param = %q, want empty", q)
	}
}
