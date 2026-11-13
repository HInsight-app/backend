package controller

import (
	"net/http"

	"backend/internal/domain"
	request "backend/internal/dto/request"
	response "backend/internal/dto/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type TagController struct {
	service domain.TagService
}

func NewTagController(e *echo.Group, svc domain.TagService) {
	ctrl := &TagController{service: svc}

	e.POST("/tags", ctrl.Create)
	e.GET("/tags", ctrl.List)
	e.POST("/tags/:id/delete", ctrl.Delete)
}

func (ctrl *TagController) Create(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, errUserContextLost)
	}

	var req request.CreateTagRequest
	if err := c.Bind(&req); err != nil {
		return errorResponse(c, http.StatusBadRequest, errInvalidPayload)
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	tag, err := ctrl.service.CreateTag(c.Request().Context(), userID, req.Name, req.ColorHex)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create tag"})
	}

	return c.JSON(http.StatusCreated, response.TagResponse{
		ID:       tag.ID,
		UserID:   tag.UserID,
		Name:     tag.Name,
		ColorHex: tag.ColorHex,
	})
}

func (ctrl *TagController) List(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, errUserContextLost)
	}

	tags, err := ctrl.service.ListTags(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch tags"})
	}

	res := make([]response.TagResponse, len(tags))
	for i, t := range tags {
		res[i] = response.TagResponse{
			ID:       t.ID,
			UserID:   t.UserID,
			Name:     t.Name,
			ColorHex: t.ColorHex,
		}
	}

	return c.JSON(http.StatusOK, res)
}

func (ctrl *TagController) Delete(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, errUserContextLost)
	}

	tagID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid tag ID"})
	}

	if err := ctrl.service.DeleteTag(c.Request().Context(), tagID, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete tag"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Tag deleted successfully"})
}
