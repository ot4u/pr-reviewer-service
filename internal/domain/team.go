package domain

import "context"

// Team представляет команду с участниками.
type Team struct {
	Name    string
	Members []*User
}

// TeamDeactivationResult представляет результат массовой деактивации
type TeamDeactivationResult struct {
	TeamName            string
	DeactivatedUsers    int
	ReassignedPRs       int
	FailedReassignments int
	DeactivatedUserIDs  []string
}

// TeamRepository определяет контракт для работы с хранилищем команд
type TeamRepository interface {
	Create(ctx context.Context, team *Team) error
	GetByName(ctx context.Context, teamName string) (*Team, error)
	GetAllUsersByTeam(ctx context.Context, teamName string) ([]*User, error)
	ExistsTeam(ctx context.Context, teamName string) (bool, error)
	GetActiveUsersFromTeam(ctx context.Context, teamName string) ([]*User, error)
	GetOpenPRsWithTeamReviewers(ctx context.Context, teamName string) ([]string, error)
	GetPRReviewersFromTeam(ctx context.Context, prID, teamName string) ([]string, error)
	GetAllTeams(ctx context.Context) ([]*Team, error)
}
