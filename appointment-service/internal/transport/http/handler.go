package http

import (
	"net/http"
	"strings"

	"appointment-service/internal/model"
	"appointment-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	uc usecase.AppointmentUseCase
}

func NewHandler(uc usecase.AppointmentUseCase) *Handler {
	return &Handler{uc: uc}
}

func statusCode(err error) int {
	msg := err.Error()
	switch {
	case strings.HasPrefix(msg, "SERVICE_UNAVAILABLE"):
		return http.StatusServiceUnavailable
	case strings.HasPrefix(msg, "DOCTOR_NOT_FOUND"):
		return http.StatusNotFound
	case strings.HasPrefix(msg, "FORBIDDEN_TRANSITION"):
		return http.StatusUnprocessableEntity
	case strings.Contains(msg, "not found"):
		return http.StatusNotFound
	default:
		return http.StatusBadRequest
	}
}

func (h *Handler) Create(c *gin.Context) {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		DoctorID    string `json:"doctor_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := h.uc.Create(req.Title, req.Description, req.DoctorID)
	if err != nil {
		c.JSON(statusCode(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, a)
}

func (h *Handler) GetByID(c *gin.Context) {
	a, err := h.uc.GetByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handler) GetAll(c *gin.Context) {
	list, err := h.uc.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) UpdateStatus(c *gin.Context) {
	var req struct {
		Status model.Status `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := h.uc.UpdateStatus(c.Param("id"), req.Status)
	if err != nil {
		c.JSON(statusCode(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, a)
}
