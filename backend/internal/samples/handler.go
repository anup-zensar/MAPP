package samples

import (
	"net/http"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

// Handler answers the HTTP requests of the developer sample page.
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the sample routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// Routes registers the sample routes on r. All of them need a signed-in person, so all are wrapped in
// signedIn. internal/app registers them only in demo mode (D-036).
func (h *Handler) Routes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/sample-folders", signedIn(h.listFolders))
	r.Handle("POST /api/sample-folders", signedIn(h.createFolder))
	r.Handle("PATCH /api/sample-folders/{folderId}", signedIn(h.updateFolder))
	r.Handle("DELETE /api/sample-folders/{folderId}", signedIn(h.deleteFolder))
	r.Handle("POST /api/sample-folders/{folderId}/notes", signedIn(h.createNote))
	r.Handle("DELETE /api/sample-notes/{noteId}", signedIn(h.deleteNote))
}

// listFolders answers GET /api/sample-folders: every folder with its notes, sorted by name.
func (h *Handler) listFolders(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	folders, err := h.service.ListFolders(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, FolderPage{Items: folders, Total: len(folders)})
}

// createFolder answers POST /api/sample-folders with 201 and the new folder.
func (h *Handler) createFolder(w http.ResponseWriter, r *http.Request) {
	var input FolderInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	folder, err := h.service.CreateFolder(r.Context(), a, input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, folder)
}

// updateFolder answers PATCH /api/sample-folders/{folderId} with the folder after the rename.
func (h *Handler) updateFolder(w http.ResponseWriter, r *http.Request) {
	var update FolderUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	folder, err := h.service.UpdateFolder(r.Context(), a, r.PathValue("folderId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, folder)
}

// deleteFolder answers DELETE /api/sample-folders/{folderId} with the recycle-bin entry of the folder and
// its notes, so the web app can offer Undo.
func (h *Handler) deleteFolder(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	entry, err := h.service.DeleteFolder(r.Context(), a, r.PathValue("folderId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// createNote answers POST /api/sample-folders/{folderId}/notes with 201 and the new note.
func (h *Handler) createNote(w http.ResponseWriter, r *http.Request) {
	var input NoteInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	note, err := h.service.CreateNote(r.Context(), a, r.PathValue("folderId"), input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, note)
}

// deleteNote answers DELETE /api/sample-notes/{noteId} with the recycle-bin entry of the note, so the web
// app can offer Undo.
func (h *Handler) deleteNote(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	entry, err := h.service.DeleteNote(r.Context(), a, r.PathValue("noteId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}
