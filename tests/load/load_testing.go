package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

const (
	targetHost = "http://localhost:8081" // e2e окружение
	rps        = 5
	duration   = 3 * time.Minute // ← теперь 3 минуты
)

type TeamMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type Team struct {
	TeamName string       `json:"team_name"`
	Members  []TeamMember `json:"members"`
}

type PRCreateRequest struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

var (
	teams []string
	users []string
	prs   []string
	httpc = &http.Client{Timeout: 10 * time.Second}
)

func postJSON(url string, body any) (int, error) {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpc.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func getURL(url string) (int, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	resp, err := httpc.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

// Seed
func seedData() error {
	log.Println("Seeding: creating teams and users...")

	for t := 1; t <= 20; t++ {
		teamName := fmt.Sprintf("team-%02d", t)
		var members []TeamMember
		for u := 1; u <= 10; u++ {
			uid := fmt.Sprintf("u-%d-%d", t, u)
			members = append(members, TeamMember{
				UserID:   uid,
				Username: fmt.Sprintf("User_%d_%d", t, u),
				IsActive: true,
			})
			users = append(users, uid)
		}

		team := Team{
			TeamName: teamName,
			Members:  members,
		}

		status, err := postJSON(targetHost+"/team/add", team)
		if err != nil {
			return err
		}
		if status >= 400 {
			log.Printf("WARN team/add returned %d\n", status)
		}

		teams = append(teams, teamName)
		time.Sleep(20 * time.Millisecond)
	}

	log.Println("Seeding: creating PRs...")

	prCounter := 1
	for _, uid := range users {
		prID := fmt.Sprintf("pr-%04d", prCounter)
		req := PRCreateRequest{
			PullRequestID:   prID,
			PullRequestName: fmt.Sprintf("PR %d by %s", prCounter, uid),
			AuthorID:        uid,
		}

		status, err := postJSON(targetHost+"/pullRequest/create", req)
		if err != nil {
			return err
		}
		if status >= 400 {
			log.Printf("WARN pullRequest/create returned %d\n", status)
		}

		prs = append(prs, prID)
		prCounter++
		time.Sleep(15 * time.Millisecond)
	}

	log.Printf("Seed completed: teams=%d users=%d prs=%d\n", len(teams), len(users), len(prs))
	return nil
}

// Targeter
func makeTargeter() vegeta.Targeter {
	return func(t *vegeta.Target) error {
		r := rand.Float64()

		// 60% GET team/get
		if r < 0.60 {
			team := teams[rand.Intn(len(teams))]
			t.Method = http.MethodGet
			t.URL = fmt.Sprintf("%s/team/get?team_name=%s", targetHost, team)
			t.Body = nil
			t.Header = map[string][]string{"Accept": {"application/json"}}
			return nil
		}

		// 35% GET users/getReview
		if r < 0.95 {
			user := users[rand.Intn(len(users))]
			t.Method = http.MethodGet
			t.URL = fmt.Sprintf("%s/users/getReview?user_id=%s", targetHost, user)
			t.Body = nil
			t.Header = map[string][]string{"Accept": {"application/json"}}
			return nil
		}

		// 3% POST team/add
		if r < 0.98 {
			teamName := fmt.Sprintf("loadteam-%d", time.Now().UnixNano())
			body, _ := json.Marshal(Team{
				TeamName: teamName,
				Members:  []TeamMember{},
			})
			t.Method = http.MethodPost
			t.URL = targetHost + "/team/add"
			t.Body = body
			t.Header = map[string][]string{"Content-Type": {"application/json"}}
			return nil
		}

		// 1.5% POST pullRequest/create
		if r < 0.995 {
			uid := users[rand.Intn(len(users))]
			prID := fmt.Sprintf("loadpr-%d", time.Now().UnixNano())
			body, _ := json.Marshal(PRCreateRequest{
				PullRequestID:   prID,
				PullRequestName: "Load PR",
				AuthorID:        uid,
			})
			t.Method = http.MethodPost
			t.URL = targetHost + "/pullRequest/create"
			t.Body = body
			t.Header = map[string][]string{"Content-Type": {"application/json"}}
			return nil
		}

		// 0.5% merge
		pr := prs[rand.Intn(len(prs))]
		body, _ := json.Marshal(map[string]string{"pull_request_id": pr})
		t.Method = http.MethodPost
		t.URL = targetHost + "/pullRequest/merge"
		t.Body = body
		t.Header = map[string][]string{"Content-Type": {"application/json"}}
		return nil
	}
}

// Attack
func runAttack() {
	rate := vegeta.Rate{Freq: rps, Per: time.Second}
	attacker := vegeta.NewAttacker()
	targeter := makeTargeter()

	var metrics vegeta.Metrics

	log.Printf("Starting attack: %s for %s", targetHost, duration)
	for res := range attacker.Attack(targeter, rate, duration, "load-test") {
		metrics.Add(res)
	}
	metrics.Close()

	fmt.Println("=== Results ===")
	fmt.Printf("Requests: %d\n", metrics.Requests)
	fmt.Printf("Success rate: %.4f%%\n", metrics.Success*100)
	fmt.Printf("Latency mean: %s\n", metrics.Latencies.Mean)
	fmt.Printf("Latency P95: %s\n", metrics.Latencies.P95)
	fmt.Printf("Latency P99: %s\n", metrics.Latencies.P99)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	if err := seedData(); err != nil {
		log.Fatalf("Seed failed: %v", err)
	}

	runAttack()
}
