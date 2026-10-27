package sample

import (
	"net/http"

	"github.com/gin-gonic/gin"

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

// RegisterRoutes registers the sample routes on r. internal/app registers them only in demo mode (D-036).
func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/sample-folders", h.listFolders)
	r.POST("/sample-folders", h.createFolder)
	r.PATCH("/sample-folders/:folderId", h.updateFolder)
	r.DELETE("/sample-folders/:folderId", h.deleteFolder)
	r.POST("/sample-folders/:folderId/notes", h.createNote)
	r.DELETE("/sample-notes/:noteId", h.deleteNote)
}

// listFolders answers GET /api/v1/sample-folders: every folder with its notes, sorted by name.
func (h *Handler) listFolders(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	folders, err := h.service.ListFolders(c.Request.Context(), a)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, FolderPage{Items: folders, Total: len(folders)})
}

// createFolder answers POST /api/v1/sample-folders with 201 and the new folder.
func (h *Handler) createFolder(c *gin.Context) {
	var input FolderInput
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	folder, err := h.service.CreateFolder(c.Request.Context(), a, input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, folder)
}

// updateFolder answers PATCH /api/v1/sample-folders/{folderId} with the folder after the rename.
func (h *Handler) updateFolder(c *gin.Context) {
	var update FolderUpdate
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	folder, err := h.service.UpdateFolder(c.Request.Context(), a, c.Param("folderId"), update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, folder)
}

// deleteFolder answers DELETE /api/v1/sample-folders/{folderId} with the recycle-bin entry of the folder and
// its notes, so the web app can offer Undo.
func (h *Handler) deleteFolder(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	entry, err := h.service.DeleteFolder(c.Request.Context(), a, c.Param("folderId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// createNote answers POST /api/v1/sample-folders/{folderId}/notes with 201 and the new note.
func (h *Handler) createNote(c *gin.Context) {
	var input NoteInput
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	note, err := h.service.CreateNote(c.Request.Context(), a, c.Param("folderId"), input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, note)
}

// deleteNote answers DELETE /api/v1/sample-notes/{noteId} with the recycle-bin entry of the note, so the web
// app can offer Undo.
func (h *Handler) deleteNote(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	entry, err := h.service.DeleteNote(c.Request.Context(), a, c.Param("noteId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}
