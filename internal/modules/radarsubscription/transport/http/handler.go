package radarsubscriptionhttp

import (
	"errors"

	radarsubscriptionapp "github.com/dujiao-next/internal/modules/radarsubscription/application"
	radarsubscriptioncontract "github.com/dujiao-next/internal/modules/radarsubscription/contract"
	"github.com/dujiao-next/internal/platform/http/ginutil"
	"github.com/dujiao-next/internal/platform/http/response"
	"github.com/gin-gonic/gin"
)

type Service interface {
	GetUserPreferences(userID uint) (*radarsubscriptionapp.UserPreferences, error)
	UpdateUserPreferences(userID uint, channels map[string]bool) (*radarsubscriptionapp.UserPreferences, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	if service == nil {
		panic("radar subscription handler: service is nil")
	}
	return &Handler{service: service}
}

func RegisterUserRoutes(group gin.IRoutes, handler *Handler) {
	group.GET("/me/radar-subscriptions", handler.GetPreferences)
	group.PUT("/me/radar-subscriptions", handler.UpdatePreferences)
}

func (h *Handler) GetPreferences(c *gin.Context) {
	uid, ok := ginutil.GetUserID(c)
	if !ok {
		return
	}
	prefs, err := h.service.GetUserPreferences(uid)
	if err != nil {
		respondRadarSubError(c, err)
		return
	}
	response.Success(c, prefs)
}

type updatePreferencesRequest struct {
	Channels map[string]bool `json:"channels" binding:"required"`
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	uid, ok := ginutil.GetUserID(c)
	if !ok {
		return
	}
	var req updatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}
	prefs, err := h.service.UpdateUserPreferences(uid, req.Channels)
	if err != nil {
		respondRadarSubError(c, err)
		return
	}
	response.Success(c, prefs)
}

func respondRadarSubError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, radarsubscriptioncontract.ErrInvalidChannel):
		ginutil.RespondError(c, response.CodeBadRequest, "error.radar_invalid_channel", nil)
	case errors.Is(err, radarsubscriptioncontract.ErrEmailBindingRequired):
		ginutil.RespondError(c, response.CodeBadRequest, "error.radar_email_required", nil)
	case errors.Is(err, radarsubscriptioncontract.ErrUserNotFound):
		ginutil.RespondError(c, response.CodeNotFound, "error.user_not_found", nil)
	default:
		ginutil.RespondError(c, response.CodeInternal, "error.internal_error", err)
	}
}
