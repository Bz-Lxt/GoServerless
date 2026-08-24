package handler

import (
	"net/http"
	"time"

	"github.com/gogo/goserverless/internal/idgen"
	"github.com/gogo/goserverless/internal/model"
	"github.com/gogo/goserverless/internal/store"
	"github.com/gogo/goserverless/internal/timeutil"
)

func newUnauthorized() *model.Error {
	return model.Unauthorized("invalid credentials or session")
}

type AuthAPI struct {
	st   *store.Store
	user string
	pass string
}

func NewAuthAPI(st *store.Store, user, pass string) *AuthAPI {
	return &AuthAPI{st: st, user: user, pass: pass}
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *AuthAPI) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	if req.Username != a.user || req.Password != a.pass {
		writeErr(w, newUnauthorized())
		return
	}
	tok := idgen.Token(24)
	exp := timeutil.NowUTC().Add(12 * time.Hour)
	if err := a.st.PutSession(r.Context(), tok, req.Username, exp); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]any{
		"token":      tok,
		"username":   req.Username,
		"expires_at": timeutil.Format(exp.In(timeutil.Beijing())),
	})
}

func (a *AuthAPI) Logout(w http.ResponseWriter, r *http.Request) {
	tok := bearer(r.Header.Get("Authorization"))
	if tok != "" {
		_ = a.st.DeleteSession(r.Context(), tok)
	}
	writeOK(w, map[string]bool{"ok": true})
}

func (a *AuthAPI) Me(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]string{"username": a.user})
}
