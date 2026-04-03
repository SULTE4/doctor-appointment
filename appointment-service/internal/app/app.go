package app

import (
	"log"
	"os"

	"appointment-service/internal/client"
	"appointment-service/internal/repository"
	handler "appointment-service/internal/transport/http"
	"appointment-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Run() {
	doctorServiceURL := os.Getenv("DOCTOR_SERVICE_URL")
	if doctorServiceURL == "" {
		doctorServiceURL = "http://localhost:8080"
	}

	repo := repository.New()
	dc := client.NewDoctorServiceClient(doctorServiceURL)
	uc := usecase.NewAppointmentUseCase(repo, dc)
	h := handler.NewAppointmentHandler(uc)

	r := gin.Default()
	r.POST("/appointments", h.Create)
	r.GET("/appointments/:id", h.GetByID)
	r.GET("/appointments", h.GetAll)
	r.PATCH("/appointments/:id/status", h.UpdateStatus)

	port := os.Getenv("APPOINTMENT-SERVICE-PORT")
	if port == "" {
		port = "8081"
	}
	log.Printf("Appointment Service listening on :%s", port)
	r.Run(":" + port)
}
