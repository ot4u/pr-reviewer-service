package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CriticalFlowsTestSuite struct {
	suite.Suite
	baseURL    string
	httpClient *http.Client
}

func (suite *CriticalFlowsTestSuite) SetupSuite() {
	suite.baseURL = "http://localhost:8081"
	suite.httpClient = &http.Client{Timeout: 10 * time.Second}
}

// generateUniqueName создает уникальное имя для теста
func (suite *CriticalFlowsTestSuite) generateUniqueName(base string) string {
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}

// createTeam создает команду и обрабатывает 409 как успех (уже существует)
func (suite *CriticalFlowsTestSuite) createTeam(teamName string, members []map[string]interface{}) error {
	teamData := map[string]interface{}{
		"team_name": teamName,
		"members":   members,
	}

	jsonData, err := json.Marshal(teamData)
	if err != nil {
		return err
	}

	resp, err := suite.httpClient.Post(suite.baseURL+"/team/add", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 201 - создано, 409 - уже существует (тоже ок)
	if resp.StatusCode != 201 && resp.StatusCode != 409 {
		return fmt.Errorf("failed to create team, status: %d", resp.StatusCode)
	}

	return nil
}

func (suite *CriticalFlowsTestSuite) TestMainFlow_CreateTeamAndPRAutoAssignment() {
	t := suite.T()

	// Используем уникальное имя команды
	teamName := suite.generateUniqueName("main-flow-team")
	teamMembers := []map[string]interface{}{
		{"user_id": "user1", "username": "User One", "is_active": true},
		{"user_id": "user2", "username": "User Two", "is_active": true},
		{"user_id": "user3", "username": "User Three", "is_active": true},
	}

	// Создаем команду
	err := suite.createTeam(teamName, teamMembers)
	assert.NoError(t, err, "Failed to create team")

	// Создаем PR
	prData := map[string]interface{}{
		"pull_request_id":   suite.generateUniqueName("pr-main"),
		"pull_request_name": "Test PR for Main Flow",
		"author_id":         "user1",
	}

	jsonData, err := json.Marshal(prData)
	assert.NoError(t, err)

	resp, err := suite.httpClient.Post(suite.baseURL+"/pullRequest/create", "application/json", bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 201, resp.StatusCode, "Failed to create PR")

	// Проверяем что PR создан
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	pr, exists := result["pr"].(map[string]interface{})
	assert.True(t, exists, "PR data not found in response")
	assert.Equal(t, "OPEN", pr["status"])
}

func (suite *CriticalFlowsTestSuite) TestMergePRFlow() {
	t := suite.T()

	teamName := suite.generateUniqueName("merge-team")
	teamMembers := []map[string]interface{}{
		{"user_id": "merge-u1", "username": "Merge User 1", "is_active": true},
		{"user_id": "merge-u2", "username": "Merge User 2", "is_active": true},
	}

	err := suite.createTeam(teamName, teamMembers)
	assert.NoError(t, err, "Failed to create team")

	// Создаем PR
	prID := suite.generateUniqueName("pr-merge")
	prData := map[string]interface{}{
		"pull_request_id":   prID,
		"pull_request_name": "Test PR for Merge",
		"author_id":         "merge-u1",
	}

	jsonData, err := json.Marshal(prData)
	assert.NoError(t, err)

	resp, err := suite.httpClient.Post(suite.baseURL+"/pullRequest/create", "application/json", bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 201, resp.StatusCode, "Failed to create PR")

	// Мерджим PR
	mergeData := map[string]interface{}{
		"pull_request_id": prID,
	}

	jsonData, err = json.Marshal(mergeData)
	assert.NoError(t, err)

	resp, err = suite.httpClient.Post(suite.baseURL+"/pullRequest/merge", "application/json", bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode, "Failed to merge PR")

	// Проверяем что PR в статусе MERGED
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	pr, exists := result["pr"].(map[string]interface{})
	assert.True(t, exists, "PR data not found in response")
	assert.Equal(t, "MERGED", pr["status"])
}

func (suite *CriticalFlowsTestSuite) TestGetUserReviewPRs() {
	t := suite.T()

	teamName := suite.generateUniqueName("user-review-team")
	teamMembers := []map[string]interface{}{
		{"user_id": "review-u1", "username": "Review User 1", "is_active": true},
		{"user_id": "review-u2", "username": "Review User 2", "is_active": true},
	}

	err := suite.createTeam(teamName, teamMembers)
	assert.NoError(t, err, "Failed to create team")

	// Создаем PR где пользователь назначен ревьювером
	prID := suite.generateUniqueName("pr-review")
	prData := map[string]interface{}{
		"pull_request_id":   prID,
		"pull_request_name": "Test PR for Review",
		"author_id":         "review-u1",
	}

	jsonData, err := json.Marshal(prData)
	assert.NoError(t, err)

	resp, err := suite.httpClient.Post(suite.baseURL+"/pullRequest/create", "application/json", bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 201, resp.StatusCode, "Failed to create PR")

	// Получаем PR пользователя
	resp, err = suite.httpClient.Get(suite.baseURL + "/users/getReview?user_id=review-u2")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode, "Failed to get user reviews")

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	// Проверяем что в ответе есть ожидаемые поля
	_, hasUserID := result["user_id"]
	_, hasPRs := result["pull_requests"]
	assert.True(t, hasUserID && hasPRs, "Response missing required fields")
}

func (suite *CriticalFlowsTestSuite) TestStatsEndpoints() {
	t := suite.T()

	// Просто проверяем что эндпоинты статистики работают
	resp, err := suite.httpClient.Get(suite.baseURL + "/stats/reviews")
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	resp, err = suite.httpClient.Get(suite.baseURL + "/stats/pr-assignments")
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
}

func TestCriticalFlowsTestSuite(t *testing.T) {
	suite.Run(t, new(CriticalFlowsTestSuite))
}
