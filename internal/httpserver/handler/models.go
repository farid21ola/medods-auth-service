package handler

import (
	"encoding/json"
	"net/http"
)

type contextKey string

const ContextKeyGUID contextKey = "guid"
const ContextKeyIP contextKey = "ip"
const ContextKeyAccessToken contextKey = "access_token"

type Response struct {
	Status string      `json:"status"`
	Msg    string      `json:"msg,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokensRequest struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type MeResponse struct {
	GUID string `json:"guid"`
}

func WriteJSONResponse(w http.ResponseWriter, statusCode int, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(resp)
}
