package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	twscraper "github.com/imperatrona/twitter-scraper"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type XClient struct {
	Scraper *twscraper.Scraper
}

func NewXClient() (client *XClient, err error) {
	cookieJSONPath := viper.GetString("x_cookie_json")
	if cookieJSONPath == "" {
		return nil, fmt.Errorf("x_cookie_json config is not set, please set it to the path of cookies.json exported from your browser")
	}

	log.Infof("==> Using cookies.json from config: %s", cookieJSONPath)
	scraper := twscraper.New()

	// Deserialize from JSON
	var cookiesJSON []CookieJSON
	f, err := os.Open(cookieJSONPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %s: %w", cookieJSONPath, err)
	}

	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close cookies.json: %w", cerr)
		}
	}()

	err = json.NewDecoder(f).Decode(&cookiesJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cookies: %w", err)
	}

	// Convert to http.Cookie
	var cookies []*http.Cookie
	var authToken, csrfToken string
	for _, c := range cookiesJSON {
		httpCookie := c.ToHTTPCookie()
		cookies = append(cookies, httpCookie)
		// Log important cookies for debugging
		if c.Name == "auth_token" {
			authToken = c.Value
			slog.Info("Found auth_token", "value", c.Value[:10]+"...")
		}
		if c.Name == "ct0" {
			csrfToken = c.Value
			slog.Info("Found ct0 (CSRF token)", "value", c.Value[:10]+"...")
		}
	}

	slog.Info("Total cookies", "count", len(cookies))
	slog.Info("Auth info", "hasAuthToken", authToken != "", "hasCSRFToken", csrfToken != "")

	// Use SetAuthToken method if we have both tokens
	if authToken != "" && csrfToken != "" {
		slog.Info("Using SetAuthToken method")
		scraper.SetAuthToken(twscraper.AuthToken{
			Token:     authToken,
			CSRFToken: csrfToken,
		})
	} else {
		slog.Info("Using SetCookies method")
		scraper.SetCookies(cookies)
	}

	// After setting Cookies or AuthToken you have to execute IsLoggedIn method.
	// Without it, scraper wouldn't be able to make requests that requires authentication
	if !scraper.IsLoggedIn() {
		return nil, fmt.Errorf("invalid cookies: failed to authenticate with X/Twitter")
	}

	return &XClient{
		Scraper: scraper,
	}, nil
}

// CookieJSON represents a cookie in JSON format (browser export format)
type CookieJSON struct {
	Name           string  `json:"name"`
	Value          string  `json:"value"`
	Path           string  `json:"path"`
	Domain         string  `json:"domain"`
	Secure         bool    `json:"secure"`
	HTTPOnly       bool    `json:"httpOnly"`
	SameSite       string  `json:"sameSite"`
	ExpirationDate float64 `json:"expirationDate,omitempty"`
	HostOnly       bool    `json:"hostOnly,omitempty"`
	Session        bool    `json:"session,omitempty"`
	StoreID        string  `json:"storeId,omitempty"`
	ID             int     `json:"id,omitempty"`
}

// ToHTTPCookie converts CookieJSON to http.Cookie
func (c *CookieJSON) ToHTTPCookie() *http.Cookie {
	cookie := &http.Cookie{
		Name:     c.Name,
		Value:    c.Value,
		Path:     c.Path,
		Domain:   c.Domain,
		Secure:   c.Secure,
		HttpOnly: c.HTTPOnly,
	}

	// Convert string SameSite to http.SameSite
	switch strings.ToLower(c.SameSite) {
	case "lax":
		cookie.SameSite = http.SameSiteLaxMode
	case "strict":
		cookie.SameSite = http.SameSiteStrictMode
	case "none", "no_restriction":
		cookie.SameSite = http.SameSiteNoneMode
	default:
		cookie.SameSite = http.SameSiteDefaultMode
	}

	return cookie
}
