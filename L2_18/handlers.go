package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type Handler struct {
	calendar *Calendar
}

func NewHandler(calendar *Calendar) *Handler {
	return &Handler{calendar: calendar}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/create_event", h.createEvent).Methods("POST")
	r.HandleFunc("/update_event", h.updateEvent).Methods("POST")
	r.HandleFunc("/delete_event", h.deleteEvent).Methods("POST")
	r.HandleFunc("/events_for_day", h.eventsForDay).Methods("GET")
	r.HandleFunc("/events_for_week", h.eventsForWeek).Methods("GET")
	r.HandleFunc("/events_for_month", h.eventsForMonth).Methods("GET")
}

type eventRequest struct {
	UserID int    `json:"user_id"`
	Date   string `json:"date"`
	Title  string `json:"title"`
	ID     int    `json:"id"`
}

type response struct {
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

func (h *Handler) createEvent(w http.ResponseWriter, r *http.Request) {
	var req eventRequest
	if err := parseRequest(r, &req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		writeError(w, "invalid date format, expected YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	event, err := h.calendar.CreateEvent(req.UserID, date, req.Title)
	if err != nil {
		writeError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	writeJSON(w, response{Result: event}, http.StatusOK)
}

func (h *Handler) updateEvent(w http.ResponseWriter, r *http.Request) {
	var req eventRequest
	if err := parseRequest(r, &req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		writeError(w, "invalid date format, expected YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	event, err := h.calendar.UpdateEvent(req.ID, req.UserID, date, req.Title)
	if err != nil {
		writeError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	writeJSON(w, response{Result: event}, http.StatusOK)
}

func (h *Handler) deleteEvent(w http.ResponseWriter, r *http.Request) {
	var req eventRequest
	if err := parseRequest(r, &req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.calendar.DeleteEvent(req.ID, req.UserID); err != nil {
		writeError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	writeJSON(w, response{Result: "event deleted"}, http.StatusOK)
}

func (h *Handler) eventsForDay(w http.ResponseWriter, r *http.Request) {
	userID, date, err := parseQueryParams(r)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	events := h.calendar.GetEventsForDay(userID, date)
	writeJSON(w, response{Result: events}, http.StatusOK)
}

func (h *Handler) eventsForWeek(w http.ResponseWriter, r *http.Request) {
	userID, date, err := parseQueryParams(r)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	events := h.calendar.GetEventsForWeek(userID, date)
	writeJSON(w, response{Result: events}, http.StatusOK)
}

func (h *Handler) eventsForMonth(w http.ResponseWriter, r *http.Request) {
	userID, date, err := parseQueryParams(r)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	events := h.calendar.GetEventsForMonth(userID, date)
	writeJSON(w, response{Result: events}, http.StatusOK)
}

func parseRequest(r *http.Request, v interface{}) error {
	if r.Header.Get("Content-Type") == "application/json" {
		return json.NewDecoder(r.Body).Decode(v)
	}

	if err := r.ParseForm(); err != nil {
		return err
	}

	// for form-urlencoded
	if idStr := r.FormValue("id"); idStr != "" {
		if id, err := strconv.Atoi(idStr); err == nil {
			v.(*eventRequest).ID = id
		}
	}
	if userIDStr := r.FormValue("user_id"); userIDStr != "" {
		if userID, err := strconv.Atoi(userIDStr); err == nil {
			v.(*eventRequest).UserID = userID
		}
	}
	v.(*eventRequest).Date = r.FormValue("date")
	v.(*eventRequest).Title = r.FormValue("title")

	return nil
}

func parseQueryParams(r *http.Request) (int, time.Time, error) {
	userIDStr := r.URL.Query().Get("user_id")
	dateStr := r.URL.Query().Get("date")

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return 0, time.Time{}, err
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0, time.Time{}, err
	}

	return userID, date, nil
}

func writeJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, errorMsg string, statusCode int) {
	writeJSON(w, response{Error: errorMsg}, statusCode)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("Completed %s in %v", r.URL.Path, time.Since(start))
	})
}
