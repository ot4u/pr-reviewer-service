package handler

import (
	"pr-reviewer-service/api"
	"pr-reviewer-service/internal/domain"

	"github.com/sirupsen/logrus"
)

type APIHandler struct {
	*TeamHandler
	*UserHandler
	*PRHandler
	*StatsHandler
}

func NewAPIHandler(
	teamUseCase domain.TeamUseCase,
	userUseCase domain.UserUseCase,
	prUseCase domain.PRUseCase,
	statsUseCase domain.StatsUseCase,
	logger *logrus.Logger,
) api.ServerInterface {

	return &APIHandler{
		TeamHandler:  NewTeamHandler(teamUseCase, logger),
		UserHandler:  NewUserHandler(userUseCase, logger),
		PRHandler:    NewPRHandler(prUseCase, logger),
		StatsHandler: NewStatsHandler(statsUseCase, logger),
	}
}
