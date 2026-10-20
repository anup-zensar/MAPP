package approvals

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/web"
)

// Handler answers the HTTP requests of approval workflows and approvals.
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the approvals routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// Routes registers the approvals routes on r. All of them need a signed-in person, so all are wrapped in
// signedIn.
func (h *Handler) Routes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/approval-workflows", signedIn(h.listWorkflows))
	r.Handle("POST /api/approval-workflows", signedIn(h.createWorkflow))
	r.Handle("GET /api/approval-workflows/{workflowId}", signedIn(h.getWorkflow))
	r.Handle("PATCH /api/approval-workflows/{workflowId}", signedIn(h.updateWorkflow))
	r.Handle("DELETE /api/approval-workflows/{workflowId}", signedIn(h.deleteWorkflow))
	r.Handle("POST /api/approval-workflows/{workflowId}/deactivate", signedIn(h.setWorkflowActive(false)))
	r.Handle("POST /api/approval-workflows/{workflowId}/activate", signedIn(h.setWorkflowActive(true)))
	r.Handle("GET /api/activities/{activityId}/approval", signedIn(h.approval))
	r.Handle("GET /api/activities/{activityId}/approval/submission", signedIn(h.submission))
	r.Handle("POST /api/activities/{activityId}/approval/submit", signedIn(h.submit))
	r.Handle("POST /api/activities/{activityId}/approval/approve", signedIn(h.decision(h.service.Approve)))
	r.Handle("POST /api/activities/{activityId}/approval/decline", signedIn(h.decision(h.service.Decline)))
	r.Handle("POST /api/activities/{activityId}/approval/recall", signedIn(h.recall))
	r.Handle("GET /api/approvals", signedIn(h.inbox))
	r.Handle("GET /api/approvals/summary", signedIn(h.summary))
	r.Handle("GET /api/approvals/activities/{activityId}", signedIn(h.preview))
	r.Handle("GET /api/approvals/monitoring", signedIn(h.monitoring))
	r.Handle("POST /api/approvals/escalations/run", signedIn(h.runEscalations))
	r.Handle("GET /api/approval-rule-options", signedIn(h.ruleOptions))
	r.Handle("GET /api/approval-delegations", signedIn(h.listDelegations))
	r.Handle("POST /api/approval-delegations", signedIn(h.createDelegation))
	r.Handle("PATCH /api/approval-delegations/{delegationId}", signedIn(h.updateDelegation))
	r.Handle("POST /api/approval-delegations/{delegationId}/cancel", signedIn(h.cancelDelegation))
	r.Handle("GET /api/notifications", signedIn(h.listNotifications))
	r.Handle("POST /api/notifications/{notificationId}/read", signedIn(h.markRead))
	r.Handle("POST /api/notifications/read-all", signedIn(h.markAllRead))
}

// listNotifications answers GET /api/notifications: the signed-in person's 50 most recent notifications
// and the unread count.
func (h *Handler) listNotifications(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	page, err := h.service.ListNotifications(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, page)
}

// markRead answers POST /api/notifications/{notificationId}/read with the notification, now read.
func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	notification, err := h.service.MarkRead(r.Context(), a, r.PathValue("notificationId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, notification)
}

// markAllRead answers POST /api/notifications/read-all with how many notifications it marked read.
func (h *Handler) markAllRead(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	result, err := h.service.MarkAllRead(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, result)
}

// decision answers POST /api/activities/{activityId}/approval/approve and …/decline, with decide being
// the service's Approve or Decline, with the Approval section after the decision.
func (h *Handler) decision(decide func(context.Context, actor.Actor, string, DecisionInput) (ActivityApproval, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input DecisionInput
		if err := web.ReadJSON(w, r, &input); err != nil {
			web.WriteError(w, r, err)
			return
		}
		a, _ := actor.From(r.Context()) // put there by the sign-in middleware
		approval, err := decide(r.Context(), a, r.PathValue("activityId"), input)
		if err != nil {
			web.WriteError(w, r, err)
			return
		}
		web.WriteJSON(w, http.StatusOK, approval)
	}
}

// approval answers GET /api/activities/{activityId}/approval: the activity's Approval section.
func (h *Handler) approval(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	approval, err := h.service.Approval(r.Context(), a, r.PathValue("activityId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, approval)
}

// submission answers GET /api/activities/{activityId}/approval/submission: what Submit would do now.
func (h *Handler) submission(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	submission, err := h.service.Submission(r.Context(), a, r.PathValue("activityId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, submission)
}

// submit answers POST /api/activities/{activityId}/approval/submit with the Approval section after the
// submission.
func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	var input SubmitInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	approval, err := h.service.Submit(r.Context(), a, r.PathValue("activityId"), input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, approval)
}

// recall answers POST /api/activities/{activityId}/approval/recall with the Approval section after the
// recall.
func (h *Handler) recall(w http.ResponseWriter, r *http.Request) {
	var input RecallInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	approval, err := h.service.Recall(r.Context(), a, r.PathValue("activityId"), input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, approval)
}

// inbox answers GET /api/approvals?tab=waiting|open|decided (default waiting): every row of the tab.
func (h *Handler) inbox(w http.ResponseWriter, r *http.Request) {
	tab := r.URL.Query().Get("tab")
	if tab == "" {
		tab = TabWaiting
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	rows, err := h.service.Rows(r.Context(), a, tab)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, RowPage{Items: rows, Total: len(rows)})
}

// summary answers GET /api/approvals/summary?monthStart&monthEnd: the counts of the summary cards.
func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var problems errs.Problems
	monthStart := instant(query, "monthStart", &problems)
	monthEnd := instant(query, "monthEnd", &problems)
	if err := problems.Err(); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	counts, err := h.service.Summary(r.Context(), a, monthStart, monthEnd)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, counts)
}

// instant reads the RFC 3339 time of the query parameter name: nil when it is absent; a value that is not
// a time adds a problem on name.
func instant(query url.Values, name string, problems *errs.Problems) *time.Time {
	text := query.Get(name)
	if text == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, text)
	if err != nil {
		problems.Add(name, reloadMessage)
		return nil
	}
	return &t
}

// preview answers GET /api/approvals/activities/{activityId}: the preview panel of the Approvals page.
func (h *Handler) preview(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	preview, err := h.service.Preview(r.Context(), a, r.PathValue("activityId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, preview)
}

// listWorkflows answers GET /api/approval-workflows: every approval workflow, sorted by name.
func (h *Handler) listWorkflows(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	workflows, err := h.service.ListWorkflows(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, WorkflowPage{Items: workflows, Total: len(workflows)})
}

// createWorkflow answers POST /api/approval-workflows with 201 and the new workflow.
func (h *Handler) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var input WorkflowInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	workflow, err := h.service.CreateWorkflow(r.Context(), a, input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, workflow)
}

// getWorkflow answers GET /api/approval-workflows/{workflowId}: the workflow and the activities using it.
func (h *Handler) getWorkflow(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	details, err := h.service.GetWorkflow(r.Context(), a, r.PathValue("workflowId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, details)
}

// updateWorkflow answers PATCH /api/approval-workflows/{workflowId} with the workflow after the change.
func (h *Handler) updateWorkflow(w http.ResponseWriter, r *http.Request) {
	var update WorkflowUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	workflow, err := h.service.UpdateWorkflow(r.Context(), a, r.PathValue("workflowId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, workflow)
}

// deleteWorkflow answers DELETE /api/approval-workflows/{workflowId} with the recycle-bin entry of the
// workflow, so the web app can offer Undo.
func (h *Handler) deleteWorkflow(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	entry, err := h.service.DeleteWorkflow(r.Context(), a, r.PathValue("workflowId"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// setWorkflowActive answers POST /api/approval-workflows/{workflowId}/deactivate (active false) and
// …/activate (active true) with the workflow after the change.
func (h *Handler) setWorkflowActive(active bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var change StatusChange
		if err := web.ReadJSON(w, r, &change); err != nil {
			web.WriteError(w, r, err)
			return
		}
		a, _ := actor.From(r.Context()) // put there by the sign-in middleware
		workflow, err := h.service.SetWorkflowActive(r.Context(), a, r.PathValue("workflowId"), active, change.Version)
		if err != nil {
			web.WriteError(w, r, err)
			return
		}
		web.WriteJSON(w, http.StatusOK, workflow)
	}
}

// monitoring answers GET /api/approvals/monitoring?monthStart&monthEnd: the Monitoring tab (017 FR-030).
func (h *Handler) monitoring(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var problems errs.Problems
	monthStart := instant(query, "monthStart", &problems)
	monthEnd := instant(query, "monthEnd", &problems)
	if err := problems.Err(); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	m, err := h.service.Monitoring(r.Context(), a, monthStart, monthEnd)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, m)
}

// runEscalations answers POST /api/approvals/escalations/run: the reminders and escalations sent now (017
// FR-028).
func (h *Handler) runEscalations(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	run, err := h.service.RunEscalations(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, run)
}

// ruleOptions answers GET /api/approval-rule-options: what a level's rule may name (017 FR-001).
func (h *Handler) ruleOptions(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	options, err := h.service.RuleOptions(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, options)
}

// listDelegations answers GET /api/approval-delegations?scope=mine|all (017 FR-024).
func (h *Handler) listDelegations(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	page, err := h.service.ListDelegations(r.Context(), a, r.URL.Query().Get("scope"))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, page)
}

// createDelegation answers POST /api/approval-delegations with 201 and the new delegation (017 FR-022).
func (h *Handler) createDelegation(w http.ResponseWriter, r *http.Request) {
	var input DelegationInput
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	view, err := h.service.CreateDelegation(r.Context(), a, input)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusCreated, view)
}

// updateDelegation answers PATCH /api/approval-delegations/{delegationId} with the delegation after the
// change (017 FR-024).
func (h *Handler) updateDelegation(w http.ResponseWriter, r *http.Request) {
	var update DelegationUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	view, err := h.service.UpdateDelegation(r.Context(), a, r.PathValue("delegationId"), update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, view)
}

// cancelDelegation answers POST /api/approval-delegations/{delegationId}/cancel with the cancelled
// delegation (017 FR-024).
func (h *Handler) cancelDelegation(w http.ResponseWriter, r *http.Request) {
	var cancel DelegationCancel
	if err := web.ReadJSON(w, r, &cancel); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	view, err := h.service.CancelDelegation(r.Context(), a, r.PathValue("delegationId"), cancel)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, view)
}
