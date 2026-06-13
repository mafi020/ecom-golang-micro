package handler

import (
	"context"

	identitypb "github.com/mafi020/ecom-golang-micro/proto/identity"

	"github.com/mafi020/ecom-golang-micro/internal/delivery/gRPC/utils"
	"github.com/mafi020/ecom-golang-micro/internal/entity"
)

// userUseCase defines the strict domain capability needed by internal RPC clients
type userUseCase interface {
	GetUserByID(ctx context.Context, id int64) (*entity.User, error)
}

type IdentityGRPCHandler struct {
	identitypb.UnimplementedIdentityServiceServer
	uc userUseCase
}

func NewIdentityGRPCHandler(uc userUseCase) *IdentityGRPCHandler {
	return &IdentityGRPCHandler{uc: uc}
}

func (h *IdentityGRPCHandler) GetUserByID(ctx context.Context, req *identitypb.GetUserRequest) (*identitypb.GetUserResponse, error) {
	user, err := h.uc.GetUserByID(ctx, req.GetId())
	if err != nil {
		return nil, utils.HandleGRPCError(err)
	}

	return &identitypb.GetUserResponse{
		User: h.toProtoUser(user),
	}, nil
}

// ── INTERNAL MAPPING LAYER ───────────────────────────────────────────────────

func (h *IdentityGRPCHandler) toProtoUser(u *entity.User) *identitypb.User {
	if u == nil {
		return nil
	}

	// Dynamic safe translation from native Role string domain to Proto enum variant
	roleEnum := identitypb.UserRole_USER_ROLE_CUSTOMER
	if u.Role == entity.RoleAdmin {
		roleEnum = identitypb.UserRole_USER_ROLE_ADMIN
	}

	return &identitypb.User{
		Id:    u.ID,
		Name:  u.Name,
		Email: u.Email,
		Role:  roleEnum.String(),
	}
}
