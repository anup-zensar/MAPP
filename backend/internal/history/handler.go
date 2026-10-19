package history

import (
	"net/http"
	"strconv"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/web"
)

// defaultLimit is how many log entries GET /api/activity-log answers when the request does not say.
const defaultLimit = 50

// Handler answers the HTTP requests about the activity log and the recycle bin. Feature 011 builds the
// screens on them; feature 001 uses Undo after a delete and the developer sample page.
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the history routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// Routes registers the history routes on r. All of them need a signed-in person, so all are wrapped in
// signedIn.
func (h *Handler) Routes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/activity-log", signedIn(h.listLog))
	r.Handle("GET /api/recycle-bin", signedIn(h.listBin))
	r.Handle("POST /api/recycle-bin/{entryId}/undo", signedIn(h.undo))
	r.Handle("POST /api/recycle-bin/{entryId}/restore", signedIn(h.restore))
}

// listLog answers GET /api/activity-log: the newest entries first, optionally only those of the record
// type recordType and of the record recordId, at most limit of them (1 to 200, 50 when not sent).
func (h *Handler) listLog(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := LogFilter{RecordType: query.Get("recordType"), RecordID: query.Get("recordId"), Limit: defaultLimit}
	if query.Has("limit") {
		limit, err := strconv.Atoi(query.Get("limit"))
		if err != nil { // not a whole number; the service checks the range of the numbers
			web.WriteError(w, r, invalidLimit())
			return
		}
		filter.Limit = limit
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	entries, total, err := h.service.ListLog(r.Context(), a, filter)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, LogPage{Items: entries, Total: total})
}

// listBin answers GET /api/recycle-bin: everything in the recycle bin, newest first, without the stored
// records themselves.
func (h *Handler) listBin(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	entries, err := h.service.ListBin(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	items := make([]BinEntryView, 0, len(entries))
	for _, e := range entries {
		items = append(items, e.View())
	}
	web.WriteJSON(w, http.StatusOK, BinPage{Items: items, Total: len(items)})
}

// undo answers POST /api/recycle-bin/{entryId}/undo: the Undo offered right after a delete (FR-026).
func (h *Handler) undo(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	result, err := h.service.Undo(r.Context(), a, r.PathValue("entryId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, result)
}

// restore answers POST /api/recycle-bin/{entryId}/restore: a restore from the recycle bin (FR-025).
func (h *Handler) restore(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	result, err := h.service.Restore(r.Context(), a, r.PathValue("entryId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, result)
}
