package invitations

import (
	"context"
	"fmt"
	"time"

	"github.com/vantaggio/prospect-api/internal/auth"
	"github.com/vantaggio/prospect-api/internal/domain"
)

type ServiceInterface interface {
	ValidateToken(ctx context.Context, token string) (*InvitationRecord, error)
	Accept(ctx context.Context, token, name, password string) (string, error)
}

type Service struct {
	repo    Repository
	authSvc auth.ServiceInterface
}

func NewService(repo Repository, authSvc auth.ServiceInterface) *Service {
	return &Service{repo: repo, authSvc: authSvc}
}

func (s *Service) ValidateToken(ctx context.Context, token string) (*InvitationRecord, error) {
	rec, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	exp, err := time.Parse(time.RFC3339, rec.ExpiresAt)
	if err == nil && time.Now().After(exp) {
		return nil, domain.ErrTokenExpired
	}
	return rec, nil
}

func (s *Service) Accept(ctx context.Context, token, name, password string) (string, error) {
	rec, err := s.ValidateToken(ctx, token)
	if err != nil {
		return "", err
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	userID, err := s.repo.CreateUserFromInvite(ctx, rec.OrgID, name, rec.Email, hash, rec.Role)
	if err != nil {
		return "", err
	}

	if err := s.repo.AcceptInvitation(ctx, token, userID); err != nil {
		return "", fmt.Errorf("mark accepted: %w", err)
	}

	pair, err := s.authSvc.Login(ctx, rec.Email, password)
	if err != nil {
		return "", fmt.Errorf("login after accept: %w", err)
	}
	return pair.AccessToken, nil
}
