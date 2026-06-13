package utils

import (
	"errors"

	"github.com/mafi020/ecom-golang-micro/internal/apperrors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HandleGRPCError converts your domain errors into concrete, standard grpc errors
func HandleGRPCError(err error) error {
	if err == nil {
		return nil
	}

	var notFound *apperrors.NotFoundError
	var conflict *apperrors.ConflictError
	var validation *apperrors.ValidationError
	var badRequest *apperrors.BadRequestError
	var unauthorized *apperrors.UnauthorizedError
	var forbidden *apperrors.ForbiddenError
	var tooManyRequests *apperrors.TooManyRequestsError

	switch {
	case errors.As(err, &notFound):
		return status.Error(codes.NotFound, err.Error())

	case errors.As(err, &conflict):
		// For maps/complex error dictionaries, we pass a summary string across standard networks
		return status.Error(codes.AlreadyExists, err.Error())

	case errors.As(err, &validation):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.As(err, &badRequest):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.As(err, &unauthorized):
		return status.Error(codes.Unauthenticated, err.Error())

	case errors.As(err, &forbidden):
		return status.Error(codes.PermissionDenied, err.Error())

	case errors.As(err, &tooManyRequests):
		return status.Error(codes.ResourceExhausted, err.Error())

	default:
		// Mask raw runtime code strings from client network vectors
		return status.Error(codes.Internal, "internal operational server error")
	}
}
