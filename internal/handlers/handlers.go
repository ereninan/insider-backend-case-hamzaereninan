package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/insider/football-league/internal/service"
)

type Handler struct {
	svc service.LeagueManager
}

func New(svc service.LeagueManager) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/api/standings", h.GetStandings)
	r.Get("/api/matches", h.GetMatches)
	r.Get("/api/week/{n}", h.GetWeekSummary)
	r.Post("/api/simulate-week", h.SimulateWeek)
	r.Post("/api/simulate-all", h.SimulateAll)
	r.Get("/api/predictions", h.GetPredictions)
	r.Put("/api/match/{id}", h.EditMatch)

	return r
}

func (h *Handler) GetStandings(w http.ResponseWriter, r *http.Request) {
	standings, err := h.svc.GetStandings()
	if err != nil {
		h.sendError(w, err.Error())
		return
	}
	h.sendJSON(w, standings)
}

func (h *Handler) SimulateWeek(w http.ResponseWriter, r *http.Request) {
	matches, err := h.svc.PlayNextWeek()
	if err != nil {
		h.sendError(w, err.Error())
		return
	}
	
	resp := map[string]interface{}{
		"played": matches,
	}
	h.sendJSON(w, resp)
}

func (h *Handler) GetMatches(w http.ResponseWriter, r *http.Request) {
	byWeek, err := h.svc.GetAllMatches()
	if err != nil {
		h.sendError(w, err.Error())
		return
	}
	h.sendJSON(w, byWeek)
}

func (h *Handler) GetWeekSummary(w http.ResponseWriter, r *http.Request) {
	nStr := chi.URLParam(r, "n")
	n, err := strconv.Atoi(nStr)
	if err != nil {
		h.sendError(w, "geçersiz hafta numarası")
		return
	}
	summary, err := h.svc.GetWeekSummary(n)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}
	h.sendJSON(w, summary)
}

func (h *Handler) SimulateAll(w http.ResponseWriter, r *http.Request) {
	matches, err := h.svc.PlayAll()
	if err != nil {
		h.sendError(w, err.Error())
		return
	}
	resp := map[string]interface{}{
		"played": matches,
	}
	h.sendJSON(w, resp)
}

func (h *Handler) GetPredictions(w http.ResponseWriter, r *http.Request) {
	predictions, err := h.svc.GetPredictions()
	if err != nil {
		h.sendError(w, err.Error())
		return
	}
	h.sendJSON(w, predictions)
}

type editRequest struct {
	HomeScore int `json:"home_score"`
	AwayScore int `json:"away_score"`
}

func (h *Handler) EditMatch(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	matchID, err := strconv.Atoi(idStr)
	if err != nil {
		h.sendError(w, "geçersiz maç id")
		return
	}

	var req editRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.sendError(w, "geçersiz json formatı")
		return
	}

	err = h.svc.EditMatchResult(matchID, req.HomeScore, req.AwayScore)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	h.sendJSON(w, map[string]string{"status": "başarılı"})
}

func (h *Handler) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) sendError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
