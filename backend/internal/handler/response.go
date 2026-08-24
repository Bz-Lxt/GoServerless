package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gogo/goserverless/internal/model"
)

type envelope struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, envelope{Data: data})
}

func writeCreated(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusCreated, envelope{Data: data})
}

func writeErr(w http.ResponseWriter, err error) {
	writeJSON(w, model.HTTPStatus(err), envelope{
		Code:    model.CodeOf(err),
		Message: model.MessageOf(err),
	})
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return model.Invalid("malformed json: " + err.Error())
	}
	return nil
}
