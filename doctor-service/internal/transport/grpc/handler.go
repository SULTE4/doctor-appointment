package grpc

import (
	"context"
	"strings"

	"doctor-service/internal/model"
	"doctor-service/internal/usecase"
	doctorpb "doctor-service/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	doctorpb.UnimplementedDoctorServiceServer
	uc usecase.DoctorUsecase
}

func NewHandler(uc usecase.DoctorUsecase) *Handler {
	return &Handler{uc: uc}
}

func toDoctorResponse(d *model.Doctor) *doctorpb.DoctorResponse {
	return &doctorpb.DoctorResponse{
		Id:             d.ID,
		FullName:       d.FullName,
		Specialization: d.Specialization,
		Email:          d.Email,
	}
}

func (h *Handler) CreateDoctor(_ context.Context, req *doctorpb.CreateDoctorRequest) (*doctorpb.DoctorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	doctor, err := h.uc.Create(strings.TrimSpace(req.GetFullName()), strings.TrimSpace(req.GetSpecialization()), strings.TrimSpace(req.GetEmail()))
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "full_name is required"), strings.Contains(err.Error(), "email is required"):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case strings.Contains(err.Error(), "email already in use"):
			return nil, status.Error(codes.AlreadyExists, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return toDoctorResponse(doctor), nil
}

func (h *Handler) GetDoctor(_ context.Context, req *doctorpb.GetDoctorRequest) (*doctorpb.DoctorResponse, error) {
	if req == nil || strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	doctor, err := h.uc.GetByID(strings.TrimSpace(req.GetId()))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return toDoctorResponse(doctor), nil
}

func (h *Handler) ListDoctors(_ context.Context, _ *doctorpb.ListDoctorsRequest) (*doctorpb.ListDoctorsResponse, error) {
	doctors, err := h.uc.GetAll()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	items := make([]*doctorpb.DoctorResponse, 0, len(doctors))
	for _, doctor := range doctors {
		items = append(items, toDoctorResponse(doctor))
	}

	return &doctorpb.ListDoctorsResponse{Doctors: items}, nil
}
