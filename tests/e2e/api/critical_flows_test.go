package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"pr-reviewer-service/api"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CriticalFlowsTestSuite struct {
	suite.Suite
	baseURL string
	client  *http.Client
}

func (suite *CriticalFlowsTestSuite) SetupSuite() {
	suite.baseURL = "http://localhost:8081"
	suite.client = &http.Client{}
}

// Каждый тест создает свои уникальные данные
func (suite *CriticalFlowsTestSuite) createTestTeam(teamName string) {
	team := api.Team{
		TeamName: teamName,
		Members: []api.TeamMember{
			{UserId: teamName + "-author", Username: teamName + " Author", IsActive: true},
			{UserId: teamName + "-reviewer1", Username: teamName + " Reviewer1", IsActive: true},
			{UserId: teamName + "-reviewer2", Username: teamName + " Reviewer2", IsActive: true},
		},
	}

	teamBody, _ := json.Marshal(team)
	resp, err := suite.client.Post(suite.baseURL+"/team/add", "application/json", bytes.NewReader(teamBody))
	if err != nil || resp.StatusCode != http.StatusCreated {
		fmt.Printf("Failed to create team %s: %v\n", teamName, err)
	}
	if resp != nil {
		resp.Body.Close()
	}
}

// Test 1: Основной flow - создание команды → создание PR → авто-назначение ревьюеров
func (suite *CriticalFlowsTestSuite) TestMainFlow_CreateTeamAndPRAutoAssignment() {
	teamName := "main-flow-team"
	suite.createTestTeam(teamName)

	// Создаем PR (должны автоматически назначиться ревьюеры)
	prRequest := api.PostPullRequestCreateJSONBody{
		PullRequestId:   "main-flow-pr",
		PullRequestName: "Main Flow Test PR",
		AuthorId:        teamName + "-author",
	}

	prBody, _ := json.Marshal(prRequest)
	resp, err := suite.client.Post(suite.baseURL+"/pullRequest/create", "application/json", bytes.NewReader(prBody))
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var prResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&prResponse)
	resp.Body.Close()

	// Проверяем что ревьюеры назначились
	pr := prResponse["pr"].(map[string]interface{})
	reviewers := pr["assigned_reviewers"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(reviewers), 1)
	assert.LessOrEqual(suite.T(), len(reviewers), 2)
}

// Test 2: Переназначение ревьюера
func (suite *CriticalFlowsTestSuite) TestReassignReviewerFlow() {
	teamName := "reassign-team"
	suite.createTestTeam(teamName)

	// Создаем PR
	prRequest := api.PostPullRequestCreateJSONBody{
		PullRequestId:   "reassign-pr",
		PullRequestName: "Reassign Test PR",
		AuthorId:        teamName + "-author",
	}

	prBody, _ := json.Marshal(prRequest)
	resp, err := suite.client.Post(suite.baseURL+"/pullRequest/create", "application/json", bytes.NewReader(prBody))
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Переназначаем ревьюера (берем первого ревьювера из команды)
	reassignRequest := api.PostPullRequestReassignJSONBody{
		PullRequestId: "reassign-pr",
		OldUserId:     teamName + "-reviewer1",
	}

	reassignBody, _ := json.Marshal(reassignRequest)
	resp, err = suite.client.Post(suite.baseURL+"/pullRequest/reassign", "application/json", bytes.NewReader(reassignBody))
	assert.NoError(suite.T(), err)

	// Может быть 200 (успех) или 409 (нет кандидатов для замены) - оба допустимы
	assert.True(suite.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusConflict,
		"Expected 200 or 409, got %d", resp.StatusCode)

	if resp != nil {
		resp.Body.Close()
	}
}

// Test 3: Мерж PR
func (suite *CriticalFlowsTestSuite) TestMergePRFlow() {
	teamName := "merge-team"
	suite.createTestTeam(teamName)

	// Создаем PR
	prRequest := api.PostPullRequestCreateJSONBody{
		PullRequestId:   "merge-pr",
		PullRequestName: "Merge Test PR",
		AuthorId:        teamName + "-author",
	}

	prBody, _ := json.Marshal(prRequest)
	resp, err := suite.client.Post(suite.baseURL+"/pullRequest/create", "application/json", bytes.NewReader(prBody))
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Мержим PR
	mergeRequest := api.PostPullRequestMergeJSONBody{
		PullRequestId: "merge-pr",
	}

	mergeBody, _ := json.Marshal(mergeRequest)
	resp, err = suite.client.Post(suite.baseURL+"/pullRequest/merge", "application/json", bytes.NewReader(mergeBody))
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

// Test 4: Получение PR пользователя
func (suite *CriticalFlowsTestSuite) TestGetUserReviewPRs() {
	teamName := "user-review-team"
	suite.createTestTeam(teamName)

	// Создаем PR где пользователь является ревьювером
	prRequest := api.PostPullRequestCreateJSONBody{
		PullRequestId:   "user-review-pr",
		PullRequestName: "User Review Test PR",
		AuthorId:        teamName + "-author",
	}

	prBody, _ := json.Marshal(prRequest)
	resp, err := suite.client.Post(suite.baseURL+"/pullRequest/create", "application/json", bytes.NewReader(prBody))
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Получаем PR пользователя
	resp, err = suite.client.Get(suite.baseURL + "/users/getReview?user_id=" + teamName + "-reviewer1")
	assert.NoError(suite.T(), err)

	// Может быть 200 (есть PR) или 404 (нет PR) - оба допустимы
	assert.True(suite.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound,
		"Expected 200 or 404, got %d", resp.StatusCode)

	if resp != nil {
		resp.Body.Close()
	}
}

// Test 5: Статистика
func (suite *CriticalFlowsTestSuite) TestStatsEndpoints() {
	// Просто проверяем что endpoints отвечают
	resp, err := suite.client.Get(suite.baseURL + "/stats/reviews")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	resp, err = suite.client.Get(suite.baseURL + "/stats/pr-assignments")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestCriticalFlowsTestSuite(t *testing.T) {
	suite.Run(t, new(CriticalFlowsTestSuite))
}
