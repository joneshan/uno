// Copyright 2018 acrazing <joking.young@gmail.com>. All rights reserved.
// Since 2018-05-24 09:06:59
// Version 1.0.0
// Desc The grpc wrapper for uno service
package uno

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type unoService struct{ *Worker }

func NewService() *unoService {
	return &unoService{NewWorker()}
}

var Service = NewService()

var _ UnoServer = NewService()

func (s *unoService) Rent(
	ctx context.Context,
	in *Empty,
) (*UnoMessage, error) {
	no := s.Worker.Rent()
	if no == 0 {
		return nil, status.Error(codes.ResourceExhausted, "the id pool is exhausted")
	}
	return &UnoMessage{No: no}, nil
}

func (s *unoService) Relet(
	ctx context.Context,
	in *UnoMessage,
) (*Empty, error) {
	if !s.Worker.Relet(in.No) {
		return nil, status.Error(codes.NotFound, "the id is not living")
	}
	return &empty, nil
}

func (s *unoService) Return(
	ctx context.Context,
	in *UnoMessage,
) (*Empty, error) {
	s.Worker.Return(in.No)
	return &empty, nil
}
