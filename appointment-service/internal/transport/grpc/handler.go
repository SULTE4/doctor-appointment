package grpc

import (
	"context"
	"strings"

	"appointment-service/internal/model"
	"appointment-service/internal/usecase"
	appointmentpb "appointment-service/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	appointmentpb.UnimplementedAppointmentServiceServer
	uc usecase.AppointmentUseCase
}

func NewHandler(uc usecase.AppointmentUseCase) *Handler {
	return &Handler{uc: uc}
}

func toAppointmentResponse(a *model.Appointment) *appointmentpb.AppointmentResponse {
	return &appointmentpb.AppointmentResponse{
		Id:          a.ID,
		Title:       a.Title,
		Description: a.Description,
		DoctorId:    a.DoctorID,
		Status:      string(a.Status),
		CreatedAt:   a.CreatedAt.Format(timeLayout),
		UpdatedAt:   a.UpdatedAt.Format(timeLayout),
	}
}

const timeLayout = "2006-01-02T15:04:05Z07:00"

func mapUseCaseError(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "title is required"),
		strings.Contains(msg, "doctor_id is required"),
		strings.Contains(msg, "status must be one of"),
		strings.Contains(msg, "FORBIDDEN_TRANSITION"):
		return status.Error(codes.InvalidArgument, msg)
	case strings.Contains(msg, "DOCTOR_NOT_FOUND"):
		return status.Error(codes.FailedPrecondition, msg)
	case strings.Contains(msg, "SERVICE_UNAVAILABLE"):
		return status.Error(codes.Unavailable, msg)
	case strings.Contains(msg, "appointment not found"):
		return status.Error(codes.NotFound, msg)
	default:
		return status.Error(codes.Internal, msg)
	}
}

func (h *Handler) CreateAppointment(ctx context.Context, req *appointmentpb.CreateAppointmentRequest) (*appointmentpb.AppointmentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	appointment, err := h.uc.Create(ctx, strings.TrimSpace(req.GetTitle()), strings.TrimSpace(req.GetDescription()), strings.TrimSpace(req.GetDoctorId()))
	if err != nil {
		return nil, mapUseCaseError(err)
	}

	return toAppointmentResponse(appointment), nil
}

func (h *Handler) GetAppointment(_ context.Context, req *appointmentpb.GetAppointmentRequest) (*appointmentpb.AppointmentResponse, error) {
	if req == nil || strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	appointment, err := h.uc.GetByID(strings.TrimSpace(req.GetId()))
	if err != nil {
		return nil, mapUseCaseError(err)
	}

	return toAppointmentResponse(appointment), nil
}

func (h *Handler) ListAppointments(_ context.Context, _ *appointmentpb.ListAppointmentsRequest) (*appointmentpb.ListAppointmentsResponse, error) {
	appointments, err := h.uc.GetAll()
	if err != nil {
		return nil, mapUseCaseError(err)
	}

	items := make([]*appointmentpb.AppointmentResponse, 0, len(appointments))
	for _, appointment := range appointments {
		items = append(items, toAppointmentResponse(appointment))
	}

	return &appointmentpb.ListAppointmentsResponse{Appointments: items}, nil
}

func (h *Handler) UpdateAppointmentStatus(ctx context.Context, req *appointmentpb.UpdateStatusRequest) (*appointmentpb.AppointmentResponse, error) {
	if req == nil || strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if strings.TrimSpace(req.GetStatus()) == "" {
		return nil, status.Error(codes.InvalidArgument, "status is required")
	}

	appointment, err := h.uc.UpdateStatus(ctx, strings.TrimSpace(req.GetId()), model.Status(strings.TrimSpace(req.GetStatus())))
	if err != nil {
		return nil, mapUseCaseError(err)
	}

	return toAppointmentResponse(appointment), nil
}
