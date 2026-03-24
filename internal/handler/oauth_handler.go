package handler

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"go-expense-tracker/internal/service"
)

type OAuthHandler struct {
	oauthService service.OAuthService
}

func NewOAuthHandler(oauthService service.OAuthService) *OAuthHandler {
	return &OAuthHandler{oauthService: oauthService}
}

// ======
// Google
// ======

func (h *OAuthHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if !h.oauthService.IsGoogleConfigured() {
		http.Error(w, "google oauth is not configured", http.StatusServiceUnavailable)
		return
	}
	state := generateStateOauthCookie(w)
	url := h.oauthService.GetGoogleLoginURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *OAuthHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if err := verifyStateOauthCookie(r); err != nil {
		http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	if code == "" {
		http.Error(w, "Code not found", http.StatusBadRequest)
		return
	}

	token, err := h.oauthService.HandleGoogleCallback(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	setAuthCookieAndRedirect(w, r, token)
}

// ======
// GitHub
// ======

func (h *OAuthHandler) HandleGitHubLogin(w http.ResponseWriter, r *http.Request) {
	if !h.oauthService.IsGitHubConfigured() {
		http.Error(w, "github oauth is not configured", http.StatusServiceUnavailable)
		return
	}
	state := generateStateOauthCookie(w)
	url := h.oauthService.GetGitHubLoginURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *OAuthHandler) HandleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	if err := verifyStateOauthCookie(r); err != nil {
		http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	if code == "" {
		http.Error(w, "Code not found", http.StatusBadRequest)
		return
	}

	token, err := h.oauthService.HandleGitHubCallback(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	setAuthCookieAndRedirect(w, r, token)
}

// gen rand string and put in HttpOnly for 10 min-s
func generateStateOauthCookie(w http.ResponseWriter) string {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	cookie := http.Cookie{
		Name:     "oauthstate",
		Value:    state,
		Expires:  time.Now().Add(10 * time.Minute),
		HttpOnly: true,
		Path:     "/",
	}
	http.SetCookie(w, &cookie)

	return state
}

// compare states from url and cookie
func verifyStateOauthCookie(r *http.Request) error {
	oauthState, err := r.Cookie("oauthstate")
	if err != nil {
		return err
	}

	if r.FormValue("state") != oauthState.Value {
		return http.ErrNoCookie
	}
	return nil
}

// redirect if token is valid
func setAuthCookieAndRedirect(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Path:     "/",
		MaxAge:   86400, // 24h
		SameSite: http.SameSiteLaxMode,
	})

	// redirect to main page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
