package httpui

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"seed-vault-release/internal/application"
	"seed-vault-release/internal/domain"
)

type errorResponse struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"requestId,omitempty"`
	} `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, code := http.StatusInternalServerError, "INTERNAL"
	message := "服务暂时无法处理请求"
	var business *domain.BusinessError
	if errors.As(err, &business) {
		code, message = string(business.Code), business.Message
		switch business.Code {
		case domain.CodeValidation:
			status = http.StatusBadRequest
		case domain.CodeConflict:
			status = http.StatusConflict
		case domain.CodeNotFound:
			status = http.StatusNotFound
		case domain.CodeForbidden:
			status = http.StatusForbidden
		case domain.CodeCorrupt:
			status = http.StatusInternalServerError
		}
	}
	var body errorResponse
	body.Error.Code, body.Error.Message, body.Error.RequestID = code, message, requestID(r)
	writeJSON(w, status, body)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return domain.Invalid("请求JSON无效: %v", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return domain.Invalid("请求只能包含一个JSON对象")
	}
	return nil
}

func enrichContext(r *http.Request, ctx *application.Context) error {
	if value := r.Header.Get("X-Actor"); value != "" {
		ctx.Actor = value
	}
	if value := r.Header.Get("X-Role"); value != "" {
		role, err := application.ParseRole(value)
		if err != nil {
			return err
		}
		ctx.Role = role
	}
	if value := r.Header.Get("Idempotency-Key"); value != "" {
		ctx.IdempotencyKey = value
	}
	return nil
}
