package client

import (
	"context"
	"fmt"
	"time"

	"appointment-service/internal/usecase"
	doctorpb "doctor-service/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type doctorServiceClient struct {
	client doctorpb.DoctorServiceClient
}

func NewDoctorServiceClient(target string) (usecase.DoctorServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to doctor service %s: %w", target, err)
	}

	return &doctorServiceClient{client: doctorpb.NewDoctorServiceClient(conn)}, conn, nil
}

func (c *doctorServiceClient) DoctorExists(ctx context.Context, doctorID string) (bool, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.client.GetDoctor(reqCtx, &doctorpb.GetDoctorRequest{Id: doctorID})
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			return false, fmt.Errorf("doctor lookup failed: %w", err)
		}

		switch st.Code() {
		case codes.NotFound:
			return false, nil
		case codes.Unavailable, codes.DeadlineExceeded:
			return false, fmt.Errorf("doctor service unavailable: %w", err)
		default:
			return false, fmt.Errorf("doctor lookup failed: %w", err)
		}
	}

	return true, nil
}
