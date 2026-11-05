package eval

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
)

type Server struct {
	cfg    Config
	store  *Store
	runner *Runner
}

func NewServer(cfg Config, store *Store) *Server {
	return &Server{cfg: cfg, store: store, runner: NewRunner(store, cfg)}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/bootstrap", s.handleBootstrap)
	mux.HandleFunc("GET /api/runs", s.handleRuns)
	mux.HandleFunc("POST /api/runs", s.handleCreateRun)
	mux.HandleFunc("GET /api/runs/{id}", s.handleRun)
	mux.HandleFunc("GET /api/compare", s.handleCompare)
	mux.HandleFunc("GET /api/case", s.handleCase)
	mux.HandleFunc("GET /api/gate", s.handleGate)
	mux.HandleFunc("POST /api/gate", s.handleGate)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
}

func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	data := s.store.View()
	datasets := make([]map[string]any, 0, len(data.Datasets))
	for _, dataset := range data.Datasets {
		datasets = append(datasets, map[string]any{
			"id": dataset.ID, "name": dataset.Name, "kind": dataset.Kind,
			"version": dataset.Version, "notes": dataset.Notes,
			"case_count": len(datasetCases(&data, dataset.ID)),
		})
	}

	slices := make([]string, 0)
	seen := map[string]bool{}
	for _, evalCase := range data.Cases {
		if !seen[evalCase.Slice] {
			seen[evalCase.Slice] = true
			slices = append(slices, evalCase.Slice)
		}
	}

	gates := append([]GateDecision(nil), data.Gates...)
	sort.Slice(gates, func(i, j int) bool { return newer(gates[i].CreatedAt, gates[i].ID, gates[j].CreatedAt, gates[j].ID) })

	writeJSON(w, http.StatusOK, map[string]any{
		"mode":       s.cfg.Mode,
		"model":      s.cfg.Model,
		"has_key":    s.cfg.APIKey != "",
		"prompts":    data.Prompts,
		"datasets":   datasets,
		"cases":      data.Cases,
		"slices":     slices,
		"runs":       s.summaries(data),
		"thresholds": data.Thresholds,
		"gates":      gates,
		"labels": map[string]any{
			"hard": HardLabels,
			"soft": softLabels,
		},
	})
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"runs": s.summaries(s.store.View())})
}

func (s *Server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PromptID  string `json:"prompt_id"`
		DatasetID string `json:"dataset_id"`
		Repeats   int    `json:"repeats"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误："+err.Error())
		return
	}
	if body.PromptID == "" || body.DatasetID == "" {
		writeError(w, http.StatusBadRequest, "必须选择提示词版本和数据集")
		return
	}

	run, err := s.runner.CreateRun(body.PromptID, body.DatasetID, body.Repeats)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	go s.runner.Start(run.ID)
	writeJSON(w, http.StatusCreated, map[string]any{"run": run})
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	data := s.store.View()
	run, ok := findRun(&data, id)
	if !ok {
		writeError(w, http.StatusNotFound, "运行记录不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"run":     run,
		"summary": SummarizeRun(data, run),
	})
}

func (s *Server) handleCompare(w http.ResponseWriter, r *http.Request) {
	data := s.store.View()
	baselineID := r.URL.Query().Get("baseline")
	candidateID := r.URL.Query().Get("candidate")

	var baseline, candidate Run
	var ok bool

	if baselineID == "" || candidateID == "" {
		done := doneRuns(data)
		if len(done) < 2 {
			writeError(w, http.StatusConflict, "至少需要两次完成的运行才能对比")
			return
		}
		candidate = done[len(done)-1]
		baseline = done[len(done)-2]
	} else {
		baseline, ok = findRun(&data, baselineID)
		if !ok {
			writeError(w, http.StatusNotFound, "基线运行不存在")
			return
		}
		candidate, ok = findRun(&data, candidateID)
		if !ok {
			writeError(w, http.StatusNotFound, "候选运行不存在")
			return
		}
	}

	payload := Compare(data, baseline, candidate, data.Thresholds)
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleCase(w http.ResponseWriter, r *http.Request) {
	data := s.store.View()
	slug := r.URL.Query().Get("slug")
	baselineID := r.URL.Query().Get("baseline")
	candidateID := r.URL.Query().Get("candidate")

	var evalCase Case
	var ok bool
	for _, item := range data.Cases {
		if item.Slug == slug || item.ID == slug {
			evalCase, ok = item, true
			break
		}
	}
	if !ok {
		writeError(w, http.StatusNotFound, "样本不存在")
		return
	}

	collect := func(runID string) map[string]any {
		var results []Result
		for _, result := range data.Results {
			if result.RunID == runID && result.CaseID == evalCase.ID {
				results = append(results, result)
			}
		}
		sort.Slice(results, func(i, j int) bool { return results[i].Repeat < results[j].Repeat })
		return map[string]any{"run_id": runID, "results": results}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"case":      evalCase,
		"baseline":  collect(baselineID),
		"candidate": collect(candidateID),
	})
}

func (s *Server) handleGate(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data := s.store.View()
		gates := append([]GateDecision(nil), data.Gates...)
		sort.Slice(gates, func(i, j int) bool { return newer(gates[i].CreatedAt, gates[i].ID, gates[j].CreatedAt, gates[j].ID) })
		writeJSON(w, http.StatusOK, map[string]any{
			"thresholds": data.Thresholds,
			"gates":      gates,
			"runs":       s.summaries(data),
		})
		return
	}

	var body struct {
		Baseline   string     `json:"baseline"`
		Candidate  string     `json:"candidate"`
		Thresholds Thresholds `json:"thresholds"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误："+err.Error())
		return
	}

	data := s.store.View()
	baseline, ok := findRun(&data, body.Baseline)
	if !ok {
		writeError(w, http.StatusNotFound, "基线运行不存在")
		return
	}
	candidate, ok := findRun(&data, body.Candidate)
	if !ok {
		writeError(w, http.StatusNotFound, "候选运行不存在")
		return
	}

	thresholds := body.Thresholds
	if thresholds.MinPassRate < 0 || thresholds.MinPassRate > 1 {
		writeError(w, http.StatusBadRequest, "最低通过率必须在 0 到 1 之间")
		return
	}
	if thresholds.MaxHardFailCases < 0 || thresholds.MaxRegressions < 0 {
		writeError(w, http.StatusBadRequest, "阈值不能为负数")
		return
	}
	if err := s.store.SetThresholds(thresholds); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	payload := Compare(data, baseline, candidate, thresholds)
	gate := GateDecision{
		ID:             newID("gate"),
		BaselineRunID:  baseline.ID,
		CandidateRunID: candidate.ID,
		Decision:       payload.Gate.Decision,
		Thresholds:     thresholds,
		Reasons:        payload.Gate.Reasons,
		CreatedAt:      nowISO(),
	}
	if err := s.store.AddGate(gate); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"gate": gate, "compare": payload})
}

func (s *Server) summaries(data Data) []RunSummary {
	runs := append([]Run(nil), data.Runs...)
	sort.Slice(runs, func(i, j int) bool { return newer(runs[i].CreatedAt, runs[i].ID, runs[j].CreatedAt, runs[j].ID) })

	result := make([]RunSummary, 0, len(runs))
	for _, run := range runs {
		result = append(result, SummarizeRun(data, run))
	}
	return result
}

func doneRuns(data Data) []Run {
	runs := append([]Run(nil), data.Runs...)
	sort.Slice(runs, func(i, j int) bool { return newer(runs[j].CreatedAt, runs[j].ID, runs[i].CreatedAt, runs[i].ID) })
	var done []Run
	for _, run := range runs {
		if run.Status == "done" {
			done = append(done, run)
		}
	}
	return done
}

func findDataset(data *Data, id string) (Dataset, bool) {
	for _, dataset := range data.Datasets {
		if dataset.ID == id {
			return dataset, true
		}
	}
	return Dataset{}, false
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	return decoder.Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": strings.TrimSpace(message)})
}

// newer 按时间倒序比较；时间相同时用 ID（含纳秒）兜底，避免同一秒内多次运行顺序抖动。
func newer(aTime, aID, bTime, bID string) bool {
	if aTime != bTime {
		return aTime > bTime
	}
	return aID > bID
}
