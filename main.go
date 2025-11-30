package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	masterListRunsPath      = "/ej/api/v1/master/list-runs-json"
	masterContestStatusPath = "/ej/api/v1/master/contest-status-json"
)

type apiClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

type apiError struct {
	LogID   string `json:"log_id"`
	Message string `json:"message"`
	Num     int    `json:"num"`
	Symbol  string `json:"symbol"`
}

type contestStatusReply struct {
	OK     bool                    `json:"ok"`
	Error  *apiError               `json:"error"`
	Result *contestStatusContainer `json:"result"`
}

type contestStatusContainer struct {
	Contest contestSummary `json:"contest"`
}

type contestSummary struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type listRunsReply struct {
	OK     bool            `json:"ok"`
	Error  *apiError       `json:"error"`
	Result *listRunsResult `json:"result"`
}

type listRunsResult struct {
	Runs         []run `json:"runs"`
	FirstRun     int   `json:"first_run"`
	LastRun      int   `json:"last_run"`
	FilteredRuns int   `json:"filtered_runs"`
}

type runRow struct {
	Contest     string `json:"contest"`
	ContestID   int    `json:"contest_id"`
	RunID       int    `json:"run_id"`
	SubmittedAt string `json:"submitted_at"`
	User        string `json:"user"`
	Problem     string `json:"problem"`
	Result      string `json:"result"`
	ContestURL  string `json:"contest_url"`
}

type run struct {
	RunID          int    `json:"run_id"`
	ContestID      int    `json:"contest_id"`
	UserLogin      string `json:"user_login"`
	UserName       string `json:"user_name"`
	ProbID         int    `json:"prob_id"`
	ProbName       string `json:"prob_name"`
	StatusStr      string `json:"status_str"`
	ScoreStr       string `json:"score_str"`
	StatusDesc     string `json:"status_desc"`
	RawScore       int    `json:"raw_score"`
	SavedScore     int    `json:"saved_score"`
	LangName       string `json:"lang_name"`
	Test           int    `json:"test"`
	TestsPassed    int    `json:"tests_passed"`
	RunTimeMillis  int    `json:"run_time"`
	SubmissionUnix int    `json:"run_time_us"`
}

func newAPIClient(baseURL, token string) *apiClient {
	return &apiClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *apiClient) get(ctx context.Context, path string, query url.Values, target any) error {
	if c.baseURL == "" {
		return errors.New("base URL is required")
	}

	fullURL, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}
	fullURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL.String(), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func (c *apiClient) fetchContestName(ctx context.Context, contestID int) (string, error) {
	params := url.Values{}
	params.Set("contest_id", strconv.Itoa(contestID))

	var reply contestStatusReply
	if err := c.get(ctx, masterContestStatusPath, params, &reply); err != nil {
		return "", err
	}
	if !reply.OK {
		return "", fmt.Errorf("contest %d: %s", contestID, formatAPIError(reply.Error))
	}
	if reply.Result == nil {
		return "", fmt.Errorf("contest %d: empty response", contestID)
	}

	name := reply.Result.Contest.Name
	if name == "" {
		name = fmt.Sprintf("contest %d", contestID)
	}
	return name, nil
}

func (c *apiClient) listRuns(ctx context.Context, contestID int, filter string, pageSize, fieldMask int) ([]run, error) {
	var allRuns []run
	first := 1

	for {
		params := url.Values{}
		params.Set("contest_id", strconv.Itoa(contestID))
		if filter != "" {
			params.Set("filter_expr", filter)
		}
		if pageSize > 0 {
			params.Set("first_run", strconv.Itoa(first))
			params.Set("last_run", strconv.Itoa(first+pageSize-1))
		}
		if fieldMask > 0 {
			params.Set("field_mask", strconv.Itoa(fieldMask))
		}

		var reply listRunsReply
		if err := c.get(ctx, masterListRunsPath, params, &reply); err != nil {
			return nil, err
		}
		if !reply.OK {
			return nil, fmt.Errorf("contest %d: %s", contestID, formatAPIError(reply.Error))
		}
		if reply.Result == nil {
			break
		}

		allRuns = append(allRuns, reply.Result.Runs...)
		if len(reply.Result.Runs) == 0 {
			break
		}

		// Stop when pagination reaches the end; fall back to size-based stopping if last_run isn't provided.
		if reply.Result.LastRun <= reply.Result.FirstRun {
			break
		}
		first = reply.Result.LastRun + 1
		if reply.Result.FilteredRuns > 0 && len(allRuns) >= reply.Result.FilteredRuns {
			break
		}
	}

	return allRuns, nil
}

func formatAPIError(err *apiError) string {
	if err == nil {
		return "unknown API error"
	}
	parts := []string{err.Message}
	if err.Symbol != "" {
		parts = append(parts, fmt.Sprintf("symbol=%s", err.Symbol))
	}
	if err.Num != 0 {
		parts = append(parts, fmt.Sprintf("num=%d", err.Num))
	}
	if err.LogID != "" {
		parts = append(parts, fmt.Sprintf("log_id=%s", err.LogID))
	}
	return strings.Join(parts, " ")
}

func parseContestIDs(commaSeparated, filePath, dirPath string) ([]int, error) {
	var ids []int
	seen := make(map[int]bool)

	addID := func(id int) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}

	if commaSeparated != "" {
		for _, token := range strings.Split(commaSeparated, ",") {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}
			id, err := strconv.Atoi(token)
			if err != nil {
				return nil, fmt.Errorf("invalid contest id %q: %w", token, err)
			}
			addID(id)
		}
	}

	if filePath != "" {
		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("open contest file: %w", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			id, err := strconv.Atoi(line)
			if err != nil {
				return nil, fmt.Errorf("invalid contest id %q in %s: %w", line, filePath, err)
			}
			addID(id)
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("read contest file: %w", err)
		}
	}

	if dirPath != "" {
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return nil, fmt.Errorf("read contest dir: %w", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if name == "" {
				continue
			}
			allDigits := true
			for _, ch := range name {
				if ch < '0' || ch > '9' {
					allDigits = false
					break
				}
			}
			if !allDigits {
				continue
			}
			id, err := strconv.Atoi(name)
			if err != nil {
				return nil, fmt.Errorf("invalid contest dir name %q: %w", name, err)
			}
			addID(id)
		}
	}

	if len(ids) == 0 {
		return nil, errors.New("no contest ids provided; use --contests, --contest-file, or --contest-dir")
	}
	return ids, nil
}

func main() {
	baseURL := flag.String("base-url", os.Getenv("EJUDGE_BASE_URL"), "Base EJUDGE URL, e.g. https://your-host")
	token := flag.String("token", os.Getenv("EJUDGE_TOKEN"), "Authorization token (also via EJUDGE_TOKEN env)")
	contestList := flag.String("contests", "", "Comma separated contest IDs")
	contestFile := flag.String("contest-file", "", "Path to file with contest IDs (one per line)")
	contestDir := flag.String("contest-dir", "", "Directory with contest folders named by numeric ID (e.g. /home/judges)")
	filterExpr := flag.String("filter", "", "Filter expression passed to list-runs")
	pageSize := flag.Int("page-size", 200, "Page size for run listing")
	fieldMask := flag.Int("field-mask", 0, "Optional field mask for list-runs")
	flag.Parse()

	ids, err := parseContestIDs(*contestList, *contestFile, *contestDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	client := newAPIClient(*baseURL, *token)
	ctx := context.Background()
	var rows []runRow

	for _, contestID := range ids {
		contestName, err := client.fetchContestName(ctx, contestID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "contest %d: %v\n", contestID, err)
			continue
		}

		runs, err := client.listRuns(ctx, contestID, *filterExpr, *pageSize, *fieldMask)
		if err != nil {
			fmt.Fprintf(os.Stderr, "contest %d: %v\n", contestID, err)
			continue
		}

		sort.Slice(runs, func(i, j int) bool {
			if runs[i].SubmissionUnix == runs[j].SubmissionUnix {
				return runs[i].RunID > runs[j].RunID
			}
			return runs[i].SubmissionUnix > runs[j].SubmissionUnix
		})

		for _, r := range runs {
			user := r.UserLogin
			if user == "" {
				user = r.UserName
			}
			prob := r.ProbName
			if prob == "" {
				prob = strconv.Itoa(r.ProbID)
			}
			status := r.StatusStr
			if status == "" {
				status = r.StatusDesc
			}
			score := r.ScoreStr
			if score == "" {
				score = fmt.Sprintf("%d", r.SavedScore)
			}
			submittedAt := ""
			if r.SubmissionUnix > 0 {
				submittedAt = time.UnixMicro(int64(r.SubmissionUnix)).Format(time.RFC3339)
			}
			contestURL := strings.TrimRight(*baseURL, "/") + fmt.Sprintf("/ej/contest/%d", contestID)
			rows = append(rows, runRow{
				Contest:     contestName,
				ContestID:   contestID,
				RunID:       r.RunID,
				SubmittedAt: submittedAt,
				User:        user,
				Problem:     prob,
				Result:      fmt.Sprintf("%s %s", status, score),
				ContestURL:  contestURL,
			})
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(rows); err != nil {
		fmt.Fprintln(os.Stderr, "encode result:", err)
		os.Exit(1)
	}
}
