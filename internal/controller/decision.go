package controller

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"backend/internal/domain"
	request "backend/internal/dto/request"
	response "backend/internal/dto/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type DecisionController struct {
	service domain.DecisionService
}

func NewDecisionController(e *echo.Group, svc domain.DecisionService) {
	ctrl := &DecisionController{service: svc}

	e.POST("/decisions", ctrl.Create)
	e.GET("/decisions", ctrl.List)
	e.GET("/decisions/:id", ctrl.GetByID)
	e.POST("/decisions/:id/update", ctrl.Update)
	e.POST("/decisions/:id/delete", ctrl.Delete)
	e.POST("/decisions/:id/sync", ctrl.Sync)

	e.POST("/decisions/:id/options", ctrl.AddOption)
	e.POST("/decisions/:id/options/:option_id/delete", ctrl.DeleteOption)

	e.POST("/decisions/:id/commit", ctrl.Commit)
	e.POST("/decisions/:id/evaluations", ctrl.CreateEvaluation)

	e.POST("/decisions/:id/tags", ctrl.AttachTag)
	e.POST("/decisions/:id/tags/:tag_id/delete", ctrl.DetachTag)
}

func (ctrl *DecisionController) Create(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, errUserContextLost)
	}

	var req request.CreateDecisionRequest
	if err := c.Bind(&req); err != nil {
		return errorResponse(c, http.StatusBadRequest, errInvalidPayload)
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	reviewDate, err := parseDate(req.ScheduledReviewDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	initialOptions := make([]domain.Option, len(req.Options))
	for i, o := range req.Options {
		initialOptions[i] = domain.Option{
			Title: o.Title,
			Pros:  o.Pros,
			Cons:  o.Cons,
		}
	}

	idempHeader := strings.TrimSpace(c.Request().Header.Get("Idempotency-Key"))
	var idempKey *string
	if idempHeader != "" {
		idempKey = &idempHeader
	}

	created, err := ctrl.service.CreateDecision(c.Request().Context(), userID, req.Title, req.Context, req.ConfidenceBefore, reviewDate, initialOptions, req.TagIDs, req.CreatedAt, idempKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, toDecisionResponse(created))
}

func (ctrl *DecisionController) List(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, errUserContextLost)
	}

	var updatedSince *time.Time
	if q := c.QueryParam("updated_since"); q != "" {
		t, err := time.Parse(time.RFC3339, q)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid updated_since timestamp format, must be RFC3339"})
		}
		updatedSince = &t
	}

	decisions, err := ctrl.service.ListDecisions(c.Request().Context(), userID, updatedSince)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch decisions"})
	}

	res := make([]response.DecisionResponse, len(decisions))
	for i, d := range decisions {
		res[i] = toDecisionResponse(d)
	}

	return c.JSON(http.StatusOK, res)
}

func (ctrl *DecisionController) GetByID(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, errUserContextLost)
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid decision ID"})
	}

	d, err := ctrl.service.GetDecision(c.Request().Context(), id, userID)
	if err != nil {
		if errors.Is(err, domain.ErrDecisionNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Decision not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, toDecisionResponse(d))
}

func (ctrl *DecisionController) Update(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, errUserContextLost)
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid decision ID"})
	}

	var req request.UpdateDecisionRequest
	if err := c.Bind(&req); err != nil {
		return errorResponse(c, http.StatusBadRequest, errInvalidPayload)
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	reviewDate, err := parseDate(req.ScheduledReviewDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	updated, err := ctrl.service.UpdateDecision(c.Request().Context(), userID, id, req.Title, req.Context, req.ConfidenceBefore, reviewDate, req.LocalUpdatedAt)
	if err != nil {
		if errors.Is(err, domain.ErrDecisionNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Decision not found"})
		}
		var conflictErr *domain.ErrConflict
		if errors.As(err, &conflictErr) {
			return c.JSON(http.StatusConflict, response.ConflictResponse{
				Error:    "Conflict: server has newer version",
				Decision: toDecisionResponse(conflictErr.ServerDecision),
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, toDecisionResponse(updated))
}

func (ctrl *DecisionController) Delete(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, errUserContextLost)
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid decision ID"})
	}

	var req request.DeleteDecisionRequest
	_ = c.Bind(&req) // Optional payload with local_updated_at

	if err := ctrl.service.DeleteDecision(c.Request().Context(), id, userID, req.LocalUpdatedAt); err != nil {
		if errors.Is(err, domain.ErrDecisionNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Decision not found"})
		}
		var conflictErr *domain.ErrConflict
		if errors.As(err, &conflictErr) {
			return c.JSON(http.StatusConflict, response.ConflictResponse{
				Error:    "Conflict: server has newer version",
				Decision: toDecisionResponse(conflictErr.ServerDecision),
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Decision deleted successfully"})
}

func (ctrl *DecisionController) Sync(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, errUserContextLost)
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid decision ID"})
	}

	var req request.SyncDecisionRequest
	if err := c.Bind(&req); err != nil {
		return errorResponse(c, http.StatusBadRequest, errInvalidPayload)
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	reviewDate, err := parseDate(req.ScheduledReviewDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	options := make([]domain.SyncOption, len(req.Options))
	for i, o := range req.Options {
		options[i] = domain.SyncOption{
			ID:        o.ID,
			ClientKey: o.ClientKey,
			Title:     o.Title,
			Pros:      o.Pros,
			Cons:      o.Cons,
		}
	}

	var outcome *domain.SyncOutcome
	if req.Outcome != nil {
		outcome = &domain.SyncOutcome{
			ChosenOptionID:  req.Outcome.ChosenOptionID,
			ChosenOptionKey: req.Outcome.ChosenOptionKey,
			CommitmentNote:  req.Outcome.CommitmentNote,
			DecidedAt:       req.Outcome.DecidedAt,
		}
	}

	data := domain.SyncDecisionData{
		Title:               req.Title,
		Context:             req.Context,
		ConfidenceBefore:    req.ConfidenceBefore,
		ScheduledReviewDate: reviewDate,
		CreatedAt:           req.CreatedAt,
		LocalUpdatedAt:      req.LocalUpdatedAt,
		TagIDs:              req.TagIDs,
		Options:             options,
		Outcome:             outcome,
	}

	synced, err := ctrl.service.SyncDecision(c.Request().Context(), id, userID, data)
	if err != nil {
		if errors.Is(err, domain.ErrDecisionNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Decision not found"})
		}
		if errors.Is(err, domain.ErrInvalidChosenOption) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		var conflictErr *domain.ErrConflict
		if errors.As(err, &conflictErr) {
			return c.JSON(http.StatusConflict, response.ConflictResponse{
				Error:    "Conflict: server has newer version",
				Decision: toDecisionResponse(conflictErr.ServerDecision),
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, toDecisionResponse(synced))
}

func (ctrl *DecisionController) AddOption(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, errUserContextLost)
	}

	decisionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid decision ID"})
	}

	var req request.CreateOptionRequest
	if err := c.Bind(&req); err != nil {
		return errorResponse(c, http.StatusBadRequest, errInvalidPayload)
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	opt, err := ctrl.service.AddOption(c.Request().Context(), decisionID, userID, req.Title, req.Pros, req.Cons)
	if err != nil {
		if errors.Is(err, domain.ErrDecisionNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Decision not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, toOptionResponse(opt))
}

func (ctrl *DecisionController) DeleteOption(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, errUserContextLost)
	}

	decisionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid decision ID"})
	}

	optionID, err := uuid.Parse(c.Param("option_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid option ID"})
	}

	if err := ctrl.service.DeleteOption(c.Request().Context(), optionID, decisionID, userID); err != nil {
		if errors.Is(err, domain.ErrCannotDeleteChosenOption) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "Cannot delete chosen option"})
		}
		if errors.Is(err, domain.ErrDecisionNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Decision not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Option deleted successfully"})
}

func (ctrl *DecisionController) Commit(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, errUserContextLost)
	}

	decisionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid decision ID"})
	}

	var req request.CommitDecisionRequest
	if err := c.Bind(&req); err != nil {
		return errorResponse(c, http.StatusBadRequest, errInvalidPayload)
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	outcome, err := ctrl.service.CommitDecision(c.Request().Context(), decisionID, userID, req.ChosenOptionID, req.CommitmentNote, req.DecidedAt)
	if err != nil {
		if errors.Is(err, domain.ErrDecisionNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Decision not found"})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, toOutcomeResponse(&outcome))
}

func (ctrl *DecisionController) CreateEvaluation(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, errUserContextLost)
	}

	decisionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid decision ID"})
	}

	var req request.CreateEvaluationRequest
	if err := c.Bind(&req); err != nil {
		return errorResponse(c, http.StatusBadRequest, errInvalidPayload)
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	eval, err := ctrl.service.CreateEvaluation(c.Request().Context(), decisionID, userID, req.SatisfactionAfter, req.ConfidenceAfter, req.HindsightNotes, req.CreatedAt)
	if err != nil {
		if errors.Is(err, domain.ErrEvaluationRequiresCommitment) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, domain.ErrDecisionNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Decision not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, toEvaluationResponse(&eval))
}

func (ctrl *DecisionController) AttachTag(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, errUserContextLost)
	}

	decisionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid decision ID"})
	}

	var req request.AttachTagRequest
	if err := c.Bind(&req); err != nil {
		return errorResponse(c, http.StatusBadRequest, errInvalidPayload)
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := ctrl.service.AttachTag(c.Request().Context(), decisionID, req.TagID, userID); err != nil {
		if errors.Is(err, domain.ErrDecisionNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Decision not found"})
		}
		if err.Error() == "tag not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Tag not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Tag attached successfully"})
}

func (ctrl *DecisionController) DetachTag(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, errUserContextLost)
	}

	decisionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid decision ID"})
	}

	tagID, err := uuid.Parse(c.Param("tag_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid tag ID"})
	}

	if err := ctrl.service.DetachTag(c.Request().Context(), decisionID, tagID, userID); err != nil {
		if errors.Is(err, domain.ErrDecisionNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Decision not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Tag detached successfully"})
}

// Helpers

func parseDate(dateStr *string) (*time.Time, error) {
	if dateStr == nil || *dateStr == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *dateStr)
	if err == nil {
		return &t, nil
	}
	t, err = time.Parse(time.RFC3339, *dateStr)
	if err == nil {
		return &t, nil
	}
	return nil, errors.New("invalid date format: must be YYYY-MM-DD or RFC3339")
}

func toDecisionResponse(d domain.Decision) response.DecisionResponse {
	var reviewDateStr *string
	if d.ScheduledReviewDate != nil {
		s := d.ScheduledReviewDate.Format("2006-01-02")
		reviewDateStr = &s
	}

	options := make([]response.OptionResponse, len(d.Options))
	for i, o := range d.Options {
		options[i] = toOptionResponse(o)
	}

	tags := make([]response.TagResponse, len(d.Tags))
	for i, t := range d.Tags {
		tags[i] = response.TagResponse{
			ID:       t.ID,
			UserID:   t.UserID,
			Name:     t.Name,
			ColorHex: t.ColorHex,
		}
	}

	status := "pending"
	if d.Evaluation != nil {
		status = "evaluated"
	} else if d.Outcome != nil {
		status = "decided"
	}

	return response.DecisionResponse{
		ID:                  d.ID,
		UserID:              d.UserID,
		Title:               d.Title,
		Status:              status,
		Context:             d.Context,
		ConfidenceBefore:    d.ConfidenceBefore,
		ScheduledReviewDate: reviewDateStr,
		CreatedAt:           d.CreatedAt,
		UpdatedAt:           d.UpdatedAt,
		ServerUpdatedAt:     d.ServerUpdatedAt,
		DeletedAt:           d.DeletedAt,
		Options:             options,
		Outcome:             toOutcomeResponse(d.Outcome),
		Evaluation:          toEvaluationResponse(d.Evaluation),
		Tags:                tags,
	}
}

func toOptionResponse(o domain.Option) response.OptionResponse {
	return response.OptionResponse{
		ID:         o.ID,
		DecisionID: o.DecisionID,
		Title:      o.Title,
		Pros:       o.Pros,
		Cons:       o.Cons,
		CreatedAt:  o.CreatedAt,
	}
}

func toOutcomeResponse(o *domain.DecisionOutcome) *response.OutcomeResponse {
	if o == nil {
		return nil
	}
	return &response.OutcomeResponse{
		DecisionID:     o.DecisionID,
		ChosenOptionID: o.ChosenOptionID,
		CommitmentNote: o.CommitmentNote,
		DecidedAt:      o.DecidedAt,
	}
}

func toEvaluationResponse(e *domain.Evaluation) *response.EvaluationResponse {
	if e == nil {
		return nil
	}
	return &response.EvaluationResponse{
		ID:                e.ID,
		DecisionID:        e.DecisionID,
		SatisfactionAfter: e.SatisfactionAfter,
		ConfidenceAfter:   e.ConfidenceAfter,
		HindsightNotes:    e.HindsightNotes,
		CreatedAt:         e.CreatedAt,
	}
}
