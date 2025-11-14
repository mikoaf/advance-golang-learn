package main

import (
	"context"
	pb "ride-sharing/shared/proto/driver"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gRPCHandler struct {
	pb.UnimplementedDriverServiceServer
	service *Service
}

func NewGRPCHandler(s *grpc.Server, service *Service) {
	handler := &gRPCHandler{
		service: service,
	}

	pb.RegisterDriverServiceServer(s, handler)
}

func (h *gRPCHandler) RegisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	driver, err := h.service.RegisterDriver(req.GetDriverID(), req.PackageSlug)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register driver: %v", err)
	}
	return &pb.RegisterDriverResponse{
		Driver: driver,
	}, nil
}

func (h *gRPCHandler) UnregisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	h.service.UnregisterDriver(req.GetDriverID())

	return &pb.RegisterDriverResponse{
		Driver: &pb.Driver{
			Id: req.GetDriverID(),
		},
	}, nil
}
