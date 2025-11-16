package usecase

import (
	"context"

	"pr-reviewer-service/internal/domain"
)

// TeamUseCase реализует бизнес-логику для работы с командами.
type TeamUseCase struct {
	teamRepo domain.TeamRepository
	userRepo domain.UserRepository
	prRepo   domain.PRRepository
}

// NewTeamUseCase создает новый экземпляр TeamUseCase.
func NewTeamUseCase(teamRepo domain.TeamRepository, userRepo domain.UserRepository, prRepo domain.PRRepository) domain.TeamUseCase {
	return &TeamUseCase{
		teamRepo: teamRepo,
		userRepo: userRepo,
		prRepo:   prRepo,
	}
}

// CreateTeam создает новую команду с участниками.
func (uc *TeamUseCase) CreateTeam(ctx context.Context, team *domain.Team) error {
	// Валидация
	if team.Name == "" {
		return domain.ErrInvalidTeamName
	}

	if len(team.Members) == 0 {
		return domain.ErrTeamMustHaveMembers
	}

	// Проверяем, что команда не существует
	exists, err := uc.teamRepo.ExistsTeam(ctx, team.Name)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrTeamAlreadyExists
	}

	// Создаем команду
	return uc.teamRepo.Create(ctx, team)
}

// GetTeam возвращает команду по названию.
func (uc *TeamUseCase) GetTeam(ctx context.Context, teamName string) (*domain.Team, error) {
	// Проверяем, что команда существует
	exists, err := uc.teamRepo.ExistsTeam(ctx, teamName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrTeamNotFound
	}

	return uc.teamRepo.GetByName(ctx, teamName)
}

// DeactivateTeamUsers массово деактивирует пользователей команды и безопасно переназначает открытые PR
func (uc *TeamUseCase) DeactivateTeamUsers(ctx context.Context, teamName string) (*domain.TeamDeactivationResult, error) {
	// Валидация
	if teamName == "" {
		return nil, domain.ErrInvalidTeamName
	}

	// Проверяем что команда существует
	exists, err := uc.teamRepo.ExistsTeam(ctx, teamName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrTeamNotFound
	}

	// Получаем активных пользователей команды
	activeUsers, err := uc.teamRepo.GetActiveUsersFromTeam(ctx, teamName)
	if err != nil {
		return nil, err
	}
	if len(activeUsers) == 0 {
		return nil, domain.ErrNoActiveUsersInTeam
	}

	// Получаем открытые PR с ревьюверами из этой команды
	openPRs, err := uc.teamRepo.GetOpenPRsWithTeamReviewers(ctx, teamName)
	if err != nil {
		return nil, err
	}

	result := &domain.TeamDeactivationResult{
		TeamName:           teamName,
		DeactivatedUserIDs: make([]string, 0, len(activeUsers)),
	}

	// Деактивируем пользователей
	for _, user := range activeUsers {
		_, err := uc.userRepo.UpdateActiveStatus(ctx, user.ID, false)
		if err != nil {
			return nil, domain.ErrTeamDeactivationFailed
		}
		result.DeactivatedUserIDs = append(result.DeactivatedUserIDs, user.ID)
	}
	result.DeactivatedUsers = len(activeUsers)

	// Безопасно переназначаем открытые PR
	for _, prID := range openPRs {
		success := uc.safelyReassignPRReviewers(ctx, prID, teamName)
		if success {
			result.ReassignedPRs++
		} else {
			result.FailedReassignments++
		}
	}

	// Если есть неудачные переназначения, возвращаем partial error
	if result.FailedReassignments > 0 {
		return result, domain.ErrPartialReassignment
	}

	return result, nil
}

// safelyReassignPRReviewers безопасно переназначает ревьюверов для PR
func (uc *TeamUseCase) safelyReassignPRReviewers(ctx context.Context, prID, teamName string) bool {
	// Получаем текущих ревьюверов из деактивируемой команды
	teamReviewers, err := uc.teamRepo.GetPRReviewersFromTeam(ctx, prID, teamName)
	if err != nil {
		return false
	}

	// Находим активных пользователей из других команд для замены
	availableReviewers, err := uc.findReplacementReviewers(ctx, teamName)
	if err != nil || len(availableReviewers) == 0 {
		return false
	}

	// Заменяем каждого ревьювера из деактивируемой команды
	for i, oldReviewerID := range teamReviewers {
		if i < len(availableReviewers) {
			newReviewerID := availableReviewers[i].ID
			err := uc.prRepo.ReassignReviewer(ctx, prID, oldReviewerID, newReviewerID)
			if err != nil {
				return false
			}
		}
	}

	return true
}

// findReplacementReviewers находит активных пользователей из других команд для замены
func (uc *TeamUseCase) findReplacementReviewers(ctx context.Context, excludeTeam string) ([]*domain.User, error) {
	// Получаем все команды кроме исключенной
	allTeams, err := uc.teamRepo.GetAllTeams(ctx) // Нужно добавить этот метод в TeamRepository
	if err != nil {
		return nil, err
	}

	var replacementReviewers []*domain.User
	for _, team := range allTeams {
		if team.Name != excludeTeam {
			activeUsers, err := uc.teamRepo.GetActiveUsersFromTeam(ctx, team.Name)
			if err != nil {
				continue
			}
			replacementReviewers = append(replacementReviewers, activeUsers...)

			// Ограничиваем количество кандидатов для производительности
			if len(replacementReviewers) >= 10 {
				break
			}
		}
	}

	return replacementReviewers, nil
}
