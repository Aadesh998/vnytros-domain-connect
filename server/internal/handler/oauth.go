package handler

import (
	"domain-connect-backend/internal/config"
	"domain-connect-backend/internal/service"
	"domain-connect-backend/internal/utils"
	"domain-connect-backend/internal/views"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strings"
)

type OAuthHandler struct {
	service service.OAuthService
}

func NewOAuthHandler(s service.OAuthService) *OAuthHandler {
	return &OAuthHandler{service: s}
}

func (h *OAuthHandler) Metadata(w http.ResponseWriter, r *http.Request) {
	utils.SendSuccess(w, h.service.Metadata())
}

func (h *OAuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req views.OAuthRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOAuthJSONError(w, &service.OAuthError{Code: "invalid_request", Description: "invalid JSON body", Status: http.StatusBadRequest})
		return
	}
	resp, err := h.service.RegisterClient(req)
	if err != nil {
		writeOAuthJSONError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *OAuthHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := views.OAuthAuthorizeParams{
		ResponseType:        q.Get("response_type"),
		ClientID:            q.Get("client_id"),
		RedirectURI:         q.Get("redirect_uri"),
		Scope:               q.Get("scope"),
		State:               q.Get("state"),
		CodeChallenge:       q.Get("code_challenge"),
		CodeChallengeMethod: q.Get("code_challenge_method"),
	}

	req, err := h.service.CreateAuthRequest(params)
	if err != nil {
		renderError(w, err)
		return
	}
	renderLogin(w, req.RequestID, "")
}

func (h *OAuthHandler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderError(w, &service.OAuthError{Code: "invalid_request", Description: "invalid form", Status: http.StatusBadRequest})
		return
	}
	requestID := r.PostForm.Get("request_id")
	email := r.PostForm.Get("email")
	password := r.PostForm.Get("password")

	if _, err := h.service.LoginPassword(requestID, email, password); err != nil {
		var oe *service.OAuthError
		if errors.As(err, &oe) && oe.Status == http.StatusUnauthorized {
			renderLogin(w, requestID, "Invalid email or password.")
			return
		}
		if errors.As(err, &oe) && oe.Status == http.StatusForbidden {
			renderLogin(w, requestID, "Your email is not verified.")
			return
		}
		renderError(w, err)
		return
	}
	renderConsent(w, requestID)
}

func (h *OAuthHandler) GoogleStart(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("request_id")
	if _, err := h.service.GetAuthRequest(requestID); err != nil {
		renderError(w, err)
		return
	}
	http.Redirect(w, r, h.service.GoogleAuthURL(requestID), http.StatusFound)
}

func (h *OAuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	requestID := strings.TrimPrefix(state, "asreq:")

	if _, err := h.service.CompleteSocialLogin(requestID, "google", code, views.GeoResponse{Country: "Unknown", City: "Unknown"}); err != nil {
		renderError(w, err)
		return
	}
	http.Redirect(w, r, "/oauth/consent?request_id="+requestID, http.StatusFound)
}

func (h *OAuthHandler) Consent(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("request_id")
	req, err := h.service.GetAuthRequest(requestID)
	if err != nil {
		renderError(w, err)
		return
	}
	if req.UserID == nil {
		renderLogin(w, requestID, "")
		return
	}
	renderConsent(w, requestID)
}

func (h *OAuthHandler) Decision(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderError(w, &service.OAuthError{Code: "invalid_request", Description: "invalid form", Status: http.StatusBadRequest})
		return
	}
	requestID := r.PostForm.Get("request_id")
	decision := r.PostForm.Get("decision")

	var redirectURL string
	var err error
	if decision == "allow" {
		redirectURL, err = h.service.IssueCode(requestID)
	} else {
		redirectURL, err = h.service.DenyRedirect(requestID)
	}
	if err != nil {
		renderError(w, err)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *OAuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthJSONError(w, &service.OAuthError{Code: "invalid_request", Description: "invalid form", Status: http.StatusBadRequest})
		return
	}
	req := views.OAuthTokenRequest{
		GrantType:    r.PostForm.Get("grant_type"),
		Code:         r.PostForm.Get("code"),
		RedirectURI:  r.PostForm.Get("redirect_uri"),
		ClientID:     r.PostForm.Get("client_id"),
		CodeVerifier: r.PostForm.Get("code_verifier"),
		RefreshToken: r.PostForm.Get("refresh_token"),
	}
	resp, err := h.service.ExchangeToken(req)
	if err != nil {
		writeOAuthJSONError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(resp)
}

func writeOAuthJSONError(w http.ResponseWriter, err error) {
	var oe *service.OAuthError
	status := http.StatusInternalServerError
	body := map[string]string{"error": "server_error"}
	if errors.As(err, &oe) {
		status = oe.Status
		body = map[string]string{"error": oe.Code, "error_description": oe.Description}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

var (
	loginTmpl   = template.Must(template.New("login").Parse(loginHTML))
	consentTmpl = template.Must(template.New("consent").Parse(consentHTML))
	errorTmpl   = template.Must(template.New("error").Parse(errorHTML))
)

func productName() string {
	if config.AppConfig != nil && config.AppConfig.ProductName != "" {
		return config.AppConfig.ProductName
	}
	return "Vnytros"
}

func renderLogin(w http.ResponseWriter, requestID, errMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := map[string]string{"RequestID": requestID, "Error": errMsg, "ProductName": productName()}
	if err := loginTmpl.Execute(w, data); err != nil {
		log.Printf("oauth: render login failed: %v", err)
	}
}

func renderConsent(w http.ResponseWriter, requestID string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := map[string]string{"RequestID": requestID, "ProductName": productName()}
	if err := consentTmpl.Execute(w, data); err != nil {
		log.Printf("oauth: render consent failed: %v", err)
	}
}

func renderError(w http.ResponseWriter, err error) {
	var oe *service.OAuthError
	status := http.StatusInternalServerError
	msg := "Something went wrong."
	if errors.As(err, &oe) {
		status = oe.Status
		msg = oe.Description
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := errorTmpl.Execute(w, map[string]string{"Message": msg}); err != nil {
		log.Printf("oauth: render error failed: %v", err)
	}
}

const baseStyle = `<style>
body{font-family:system-ui,-apple-system,Segoe UI,Roboto,sans-serif;background:#0f1115;color:#e7e9ee;display:flex;min-height:100vh;align-items:center;justify-content:center;margin:0}
.card{background:#171a21;border:1px solid #262b36;border-radius:14px;padding:32px;width:360px;box-shadow:0 10px 40px rgba(0,0,0,.4)}
h1{font-size:18px;margin:0 0 4px}p.sub{color:#9aa3b2;font-size:13px;margin:0 0 20px}
label{display:block;font-size:12px;color:#9aa3b2;margin:14px 0 6px}
input{width:100%;box-sizing:border-box;padding:10px 12px;border-radius:8px;border:1px solid #2b313d;background:#0f1115;color:#e7e9ee;font-size:14px}
button{width:100%;margin-top:18px;padding:11px;border:0;border-radius:8px;background:#4f7cff;color:#fff;font-size:14px;font-weight:600;cursor:pointer}
button.secondary{background:#252b36}
.row{display:flex;gap:10px}.row button{margin-top:0}
.social{display:block;text-align:center;margin-top:10px;padding:10px;border-radius:8px;border:1px solid #2b313d;color:#e7e9ee;text-decoration:none;font-size:14px}
.err{background:#3a1d22;border:1px solid #5b2b33;color:#ffb4bd;padding:10px;border-radius:8px;font-size:13px;margin-bottom:8px}
.hr{display:flex;align-items:center;color:#6b7480;font-size:12px;margin:18px 0}.hr:before,.hr:after{content:"";flex:1;height:1px;background:#262b36;margin:0 10px}
</style>`

const loginHTML = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Sign in</title>` + baseStyle + `</head><body>
<div class="card">
  <h1>Sign in</h1>
  <p class="sub">Authorize access to your {{.ProductName}} account.</p>
  {{if .Error}}<div class="err">{{.Error}}</div>{{end}}
  <form method="post" action="/oauth/login">
    <input type="hidden" name="request_id" value="{{.RequestID}}">
    <label>Email</label>
    <input type="email" name="email" required autofocus>
    <label>Password</label>
    <input type="password" name="password" required>
    <button type="submit">Continue</button>
  </form>
  <div class="hr">or</div>
  <a class="social" href="/oauth/authorize/google?request_id={{.RequestID}}">Continue with Google</a>
</div></body></html>`

const consentHTML = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Authorize</title>` + baseStyle + `</head><body>
<div class="card">
  <h1>Authorize access</h1>
  <p class="sub">An application is requesting access to your {{.ProductName}} account, including your linked domains.</p>
  <form method="post" action="/oauth/decision">
    <input type="hidden" name="request_id" value="{{.RequestID}}">
    <div class="row">
      <button class="secondary" type="submit" name="decision" value="deny">Deny</button>
      <button type="submit" name="decision" value="allow">Allow</button>
    </div>
  </form>
</div></body></html>`

const errorHTML = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Error</title>` + baseStyle + `</head><body>
<div class="card">
  <h1>Request failed</h1>
  <p class="sub">{{.Message}}</p>
</div></body></html>`
