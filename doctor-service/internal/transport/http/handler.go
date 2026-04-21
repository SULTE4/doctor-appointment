package http

import (
	"net/http"
	"strings"

	"doctor-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	uc usecase.DoctorUsecase
}

func NewHandler(uc usecase.DoctorUsecase) *Handler {
	return &Handler{uc: uc}
}

func statusCode(err error) int {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "email already in use"):
		return http.StatusConflict
	case strings.Contains(msg, "not found"):
		return http.StatusNotFound
	default:
		return http.StatusBadRequest
	}
}

func (h *Handler) Create(c *gin.Context) {
	var req struct {
		FullName string `json:"full_name" binding:"required"`
		Spec     string `json:"specialization"`
		Email    string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	doc, err := h.uc.Create(req.FullName, req.Spec, req.Email)
	if err != nil {
		c.JSON(statusCode(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"doctor": doc})
}

func (h *Handler) GetAll(c *gin.Context) {
	docs, err := h.uc.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"doctors": docs})
}

func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	doc, err := h.uc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"doctor": doc})
}
