package app

import (
	"fmt"
	"log"
	"net"
	"os"

	"doctor-service/internal/repository"
	grpcHandler "doctor-service/internal/transport/grpc"
	"doctor-service/internal/usecase"
	doctorpb "doctor-service/proto"

	"google.golang.org/grpc"
)

func Run() {
	port := os.Getenv("DOCTOR_SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	repo := repository.New()
	uc := usecase.New(repo)
	h := grpcHandler.NewHandler(uc)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("[FATAL] failed to listen on :%s: %v", port, err)
	}

	server := grpc.NewServer()
	doctorpb.RegisterDoctorServiceServer(server, h)

	log.Printf("[INFO] Doctor Service gRPC listening on :%s", port)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("[FATAL] failed to serve doctor gRPC server: %v", err)
	}
}
