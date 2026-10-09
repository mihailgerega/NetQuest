package auth

import (
	"context"
	"fmt"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
	"github.com/netquest/netquest/backend/pkg/idgen"
)

// Данные общего demo-аккаунта.
const (
	demoUserSeed    = "netquest-demo-user" // из него строится детерминированный ID
	demoEmail       = "demo@netquest.local"
	demoDisplayName = "NetQuest Demo"
)

// Demo входит в общий demo-аккаунт без пароля: чтобы попробовать NetQuest,
// регистрация не нужна. Аккаунт один на всех и создаётся при первом demo-входе.
//
// Выключается настройкой DEMO_AUTH_ENABLED → ErrDemoAuthDisabled (403).
func (s *service) Demo(ctx context.Context, client input.ClientInfo) (model.AuthTokens, error) {
	if !s.settings.DemoAuthEnabled {
		return model.AuthTokens{}, errs.ErrDemoAuthDisabled
	}

	user, err := s.userRepository.UpsertDemoUser(ctx, model.User{
		ID:          idgen.DeterministicUUID(demoUserSeed),
		Email:       demoEmail,
		DisplayName: demoDisplayName,
		Role:        model.RoleUser,
	})
	if err != nil {
		return model.AuthTokens{}, fmt.Errorf("подготовить demo-аккаунт: %w", err)
	}

	return s.issueTokens(ctx, user, client)
}
