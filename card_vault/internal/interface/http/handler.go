package httpinterface

import (
	"encoding/json"
	"errors"
	"huhnak/card_vault/internal/application"
	"huhnak/card_vault/internal/domain"
	"net/http"
)

type Handler struct {
	service *application.CardService
}

func NewHandler(service *application.CardService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRouter(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/cards/tokenize", h.handleTokenize)
}

func (h *Handler) handleTokenize(w http.ResponseWriter, r *http.Request) {
	var input domain.CardInput

	r.Body = http.MaxBytesReader(w, r.Body, 1024*10)

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeErrorResponse(&w, http.StatusBadRequest, "invalid json body")
		return
	}

	card, err := h.service.TokenizeCard(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCardExpired),
			errors.Is(err, domain.ErrCardNotFound),
			errors.Is(err, domain.ErrInvalidCVV),
			errors.Is(err, domain.ErrInvalidCardNumber):
			writeErrorResponse(&w, http.StatusBadRequest, err.Error())
		default:
			writeErrorResponse(&w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeOkResponse(&w, http.StatusCreated, card)
}

func (h *Handler) handleCardByToken(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	if token == "" {
		writeErrorResponse(&w, http.StatusBadRequest, "token is required")
	}
	card, err := h.service.GetById(r.Context(), token)
	if err != nil {
		if errors.Is(err, domain.ErrCardNotFound) {
			writeErrorResponse(&w, http.StatusNotFound, "token card not found")
			return
		}
		writeErrorResponse(&w, http.StatusInternalServerError, "internal server error")
	}
	writeOkResponse(&w, http.StatusOK, card)
}

func writeErrorResponse(w *http.ResponseWriter, statusCode int, errorMessage string) {
	(*w).Header().Set("Content-Type", "application/json")
	(*w).WriteHeader(statusCode)
	_ = json.NewEncoder(*w).Encode(map[string]string{"error": errorMessage})
}
func writeOkResponse(w *http.ResponseWriter, statusCode int, responseObject any) {
	(*w).Header().Set("Content-Type", "application/json")
	(*w).WriteHeader(statusCode)
	_ = json.NewEncoder(*w).Encode(responseObject)
}
