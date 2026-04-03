package app

import (
	"fmt"
	"log"
	"os"

	"doctor-service/internal/repository"
	handler "doctor-service/internal/transport/http"
	"doctor-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Run() {
	port := os.Getenv("DOCTOR_SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	repo := repository.New()
	uc := usecase.New(repo)
	h := handler.NewHandler(uc)

	r := gin.Default()
	r.POST("/doctors", h.Create)
	r.GET("/doctors", h.GetAll)
	r.GET("/doctors/:id", h.GetByID)

	log.Printf("[INFO] Doctor Service starting on :%s", port)
	r.Run(fmt.Sprintf(":%s", port))
}
