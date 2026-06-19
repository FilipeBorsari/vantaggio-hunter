// Package invitations gerencia o fluxo de convite de usuários:
// org_admin cria convites (via orgadmin); este pacote expõe
// os endpoints públicos (sem auth) de validação e aceite do token.
// Ao aceitar, o usuário é criado e logado na mesma transação.
package invitations

import "context"

type Repository interface {
	FindByToken(ctx context.Context, token string) (*InvitationRecord, error)
	AcceptInvitation(ctx context.Context, token, userID string) error
	CreateUserFromInvite(ctx context.Context, orgID, name, email, passwordHash, role string) (string, error)
}

type InvitationRecord struct {
	ID        string
	OrgID     string
	OrgName   string
	Email     string
	Role      string
	Token     string
	ExpiresAt string
}
