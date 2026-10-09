package routes_auth

import (
	"aura/config"
	"aura/logging"
	"aura/utils/httpx"
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const oidcStateCookieName = "aura_oidc_state"
const oidcStateTTLSeconds = 5 * 60

var (
	oidcProvider   *oidc.Provider
	oidcOAuth2Cfg  *oauth2.Config
	oidcIDVerifier *oidc.IDTokenVerifier
)

// InitOIDC performs OIDC discovery against the configured issuer and, on success, makes OIDC
// login available. If discovery fails (unreachable/misconfigured IdP), OIDC is simply left
// unavailable - the rest of the app must keep working, since a broken IdP shouldn't take down
// the whole server. The error is returned so callers can show why OIDC is unavailable.
func InitOIDC(ctx context.Context) (Err logging.LogErrorInfo) {
	oidcProvider = nil
	oidcOAuth2Cfg = nil
	oidcIDVerifier = nil

	cfg := config.Current.Auth.OIDC
	if !cfg.Enabled {
		return logging.LogErrorInfo{}
	}

	// The Redirect URL must point to aura's OIDC callback, otherwise the login can never complete
	redirectURL, err := url.Parse(cfg.RedirectURL)
	if err != nil || !strings.HasSuffix(strings.TrimRight(redirectURL.Path, "/"), "/api/auth/oidc/callback") {
		logging.LOGGER.Error().Timestamp().Str("redirect_url", cfg.RedirectURL).Msg("OIDC Redirect URL does not point to aura's OIDC callback - OIDC login will be unavailable until this is fixed")
		return logging.LogErrorInfo{
			Message: "OIDC Redirect URL must point to aura's OIDC callback",
			Help:    "Set it to your aura URL followed by /api/auth/oidc/callback (e.g. https://aura.example.com/api/auth/oidc/callback), and register the same URL with your identity provider",
			Detail:  map[string]any{"redirect_url": cfg.RedirectURL},
		}
	}

	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		logging.LOGGER.Error().Timestamp().Str("issuer_url", cfg.IssuerURL).Err(err).Msg("OIDC discovery failed - OIDC login will be unavailable until this is fixed and the config is reloaded")
		return logging.LogErrorInfo{
			Message: "OIDC discovery failed",
			Help:    "Ensure the Issuer URL is the base URL of your identity provider (e.g. https://idp.example.com) and that aura can reach it",
			Detail:  map[string]any{"issuer_url": cfg.IssuerURL, "error": err.Error()},
		}
	}

	oidcProvider = provider
	oidcOAuth2Cfg = &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	oidcIDVerifier = provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})

	logging.LOGGER.Info().Timestamp().Str("issuer_url", cfg.IssuerURL).Msg("OIDC initialized successfully")
	return logging.LogErrorInfo{}
}

// OIDCAvailable reports whether OIDC login is configured, enabled, and successfully initialized.
func OIDCAvailable() bool {
	return config.Current.Auth.OIDC.Enabled && oidcProvider != nil && oidcOAuth2Cfg != nil && oidcIDVerifier != nil
}

// OIDCLoginRedirect godoc
// @Summary      Start OIDC Login
// @Description  Redirects to the configured OIDC identity provider to start a browser login. Not usable programmatically - this is a browser redirect flow.
// @Tags         Auth
// @Success      302
// @Router       /api/auth/oidc/login [get]
func OIDCLoginRedirect(w http.ResponseWriter, r *http.Request) {
	if !OIDCAvailable() {
		http.Redirect(w, r, "/login?error=oidc_unavailable", http.StatusFound)
		return
	}

	state, err := randomHexString(32)
	if err != nil {
		http.Redirect(w, r, "/login?error=oidc_failed", http.StatusFound)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookieName,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   resolveSecure(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   oidcStateTTLSeconds,
	})

	http.Redirect(w, r, oidcOAuth2Cfg.AuthCodeURL(state), http.StatusFound)
}

// OIDCCallback godoc
// @Summary      OIDC Callback
// @Description  Handles the redirect back from the OIDC identity provider, exchanges the authorization code, verifies the ID token, and - if the authenticated identity is allowed - starts a browser session the same way password login does.
// @Tags         Auth
// @Success      302
// @Router       /api/auth/oidc/callback [get]
func OIDCCallback(w http.ResponseWriter, r *http.Request) {
	ctx, ld := logging.CreateLoggingContext(r.Context(), r.URL.Path)
	logAction := ld.AddAction("OIDC Callback", logging.LevelInfo)
	ctx = logging.WithCurrentAction(ctx, logAction)
	defer logAction.Complete()

	clearOIDCStateCookie(w, r)

	if !OIDCAvailable() {
		http.Redirect(w, r, "/login?error=oidc_unavailable", http.StatusFound)
		return
	}

	stateCookie, err := r.Cookie(oidcStateCookieName)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		logAction.SetError("OIDC state mismatch", "The state parameter did not match - the login attempt may have expired or been tampered with", nil)
		http.Redirect(w, r, "/login?error=oidc_failed", http.StatusFound)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		logAction.SetError("OIDC callback missing code", "", nil)
		http.Redirect(w, r, "/login?error=oidc_failed", http.StatusFound)
		return
	}

	oauth2Token, err := oidcOAuth2Cfg.Exchange(ctx, code)
	if err != nil {
		logAction.SetError("OIDC code exchange failed", err.Error(), nil)
		http.Redirect(w, r, "/login?error=oidc_failed", http.StatusFound)
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		logAction.SetError("OIDC token response missing id_token", "", nil)
		http.Redirect(w, r, "/login?error=oidc_failed", http.StatusFound)
		return
	}

	idToken, err := oidcIDVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		logAction.SetError("OIDC ID token verification failed", err.Error(), nil)
		http.Redirect(w, r, "/login?error=oidc_failed", http.StatusFound)
		return
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		logAction.SetError("Failed to parse OIDC claims", err.Error(), nil)
		http.Redirect(w, r, "/login?error=oidc_failed", http.StatusFound)
		return
	}

	if !isOIDCIdentityAllowed(claims.Email) {
		logAction.SetError("OIDC identity not allowed", "The authenticated email is not in the configured allowed emails/domains list", map[string]any{"email": claims.Email})
		http.Redirect(w, r, "/login?error=oidc_not_allowed", http.StatusFound)
		return
	}

	if err := IssueSessionCookie(w, r); err != nil {
		logAction.SetError("Failed to start session", err.Error(), nil)
		http.Redirect(w, r, "/login?error=oidc_failed", http.StatusFound)
		return
	}

	logAction.AppendResult("oidc_login_success", claims.Email)
	http.Redirect(w, r, "/", http.StatusFound)
}

// isOIDCIdentityAllowed checks the authenticated email against the configured allowlists.
// If both AllowedEmails and AllowedDomains are empty, any successfully authenticated IdP user
// is allowed - this is surfaced as a loud warning in the Settings UI, not hidden here.
func isOIDCIdentityAllowed(email string) bool {
	cfg := config.Current.Auth.OIDC
	if len(cfg.AllowedEmails) == 0 && len(cfg.AllowedDomains) == 0 {
		return true
	}

	emailLower := strings.ToLower(strings.TrimSpace(email))
	if emailLower == "" {
		return false
	}

	for _, allowed := range cfg.AllowedEmails {
		if strings.ToLower(strings.TrimSpace(allowed)) == emailLower {
			return true
		}
	}

	domain := ""
	if idx := strings.LastIndex(emailLower, "@"); idx != -1 {
		domain = emailLower[idx+1:]
	}
	for _, allowedDomain := range cfg.AllowedDomains {
		if strings.ToLower(strings.TrimSpace(allowedDomain)) == domain {
			return true
		}
	}

	return false
}

func clearOIDCStateCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   resolveSecure(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func randomHexString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type authMethodsResponse struct {
	PasswordEnabled bool `json:"password_enabled"`
	OIDCEnabled     bool `json:"oidc_enabled"`
}

// GetAuthMethods godoc
// @Summary      Get Available Auth Methods
// @Description  Public, minimal endpoint the login page uses to know which login methods to show (password / OIDC SSO) before the user is authenticated.
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  httpx.JSONResponse{data=authMethodsResponse}
// @Router       /api/config/auth-methods [get]
func GetAuthMethods(w http.ResponseWriter, r *http.Request) {
	_, ld := logging.CreateLoggingContext(r.Context(), r.URL.Path)
	httpx.SendResponse(w, ld, authMethodsResponse{
		PasswordEnabled: config.Current.Auth.Enabled,
		OIDCEnabled:     OIDCAvailable(),
	})
}
