package app

import (
	"fmt"
	"log"
	"net"
	"os"

	"appointment-service/internal/client"
	"appointment-service/internal/repository"
	grpcHandler "appointment-service/internal/transport/grpc"
	"appointment-service/internal/usecase"
	appointmentpb "appointment-service/proto"

	"google.golang.org/grpc"
)

func Run() {
	doctorServiceAddress := os.Getenv("DOCTOR_SERVICE_ADDR")
	if doctorServiceAddress == "" {
		doctorServiceAddress = "localhost:8080"
	}

	repo := repository.New()
	dc, conn, err := client.NewDoctorServiceClient(doctorServiceAddress)
	if err != nil {
		log.Fatalf("[FATAL] failed to initialize doctor gRPC client: %v", err)
	}
	defer conn.Close()

	uc := usecase.New(repo, dc)
	h := grpcHandler.NewHandler(uc)

	port := os.Getenv("APPOINTMENT_SERVICE_PORT")
	if port == "" {
		port = "8081"
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("[FATAL] failed to listen on :%s: %v", port, err)
	}

	server := grpc.NewServer()
	appointmentpb.RegisterAppointmentServiceServer(server, h)

	log.Printf("[INFO] Appointment Service gRPC listening on :%s", port)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("[FATAL] failed to serve appointment gRPC server: %v", err)
	}
}
