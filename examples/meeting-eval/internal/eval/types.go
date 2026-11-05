package eval

// 领域模型：提示词、测试集、运行、结果、门禁。
// 所有会影响结论的输入都做快照，避免“这次跑的是哪版”说不清。

type Prompt struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Body      string `json:"body"`
	Notes     string `json:"notes"`
	CreatedAt string `json:"created_at"`
}

type Dataset struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"` // dev | holdout
	Version   string `json:"version"`
	Notes     string `json:"notes"`
	CreatedAt string `json:"created_at"`
}

type Expectation struct {
	Items      []ExpectationItem `json:"items"`
	AllowExtra bool              `json:"allow_extra"`
	NoAction   bool              `json:"no_action"`
}

type ExpectationItem struct {
	Contains []string `json:"contains"`
	Exclude  []string `json:"exclude,omitempty"`
	Owner    *string  `json:"owner"`
	Due      *string  `json:"due"`
	Status   string   `json:"status"`
}

type Case struct {
	ID          string      `json:"id"`
	DatasetID   string      `json:"dataset_id"`
	Slug        string      `json:"slug"`
	Title       string      `json:"title"`
	Slice       string      `json:"slice"`
	MeetingDate string      `json:"meeting_date"`
	Transcript  string      `json:"transcript"`
	Expect      Expectation `json:"expect"`
	Note        string      `json:"note"`
	OrderIndex  int         `json:"order_index"`
}

type Run struct {
	ID          string  `json:"id"`
	PromptID    string  `json:"prompt_id"`
	DatasetID   string  `json:"dataset_id"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	Mode        string  `json:"mode"` // live | replay
	Repeats     int     `json:"repeats"`
	Status      string  `json:"status"` // queued | running | done | error
	TotalCases  int     `json:"total_cases"`
	DoneCases   int     `json:"done_cases"`
	Error       string  `json:"error,omitempty"`
	CreatedAt   string  `json:"created_at"`
	FinishedAt  string  `json:"finished_at,omitempty"`
}

type Finding struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Item    *int   `json:"item,omitempty"`
}

type Check struct {
	Label  string `json:"label"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type RuleResult struct {
	Checks     []Check   `json:"checks"`
	HardFails  []Finding `json:"hard_fails"`
	SoftIssues []Finding `json:"soft_issues"`
	Quality    int       `json:"quality"`
	Matched    int       `json:"matched"`
}

type ModelItem struct {
	Task     string `json:"task"`
	Owner    string `json:"owner"`
	Due      string `json:"due"`
	Status   string `json:"status"`
	Evidence string `json:"evidence"`
}

type ModelOutput struct {
	ActionItems []ModelItem `json:"action_items"`
}

type JudgeItem struct {
	Index         int     `json:"index"`
	Supported     *bool   `json:"supported,omitempty"`
	EvidenceOK    *bool   `json:"evidence_ok,omitempty"`
	OwnerOK       *bool   `json:"owner_ok,omitempty"`
	DueOK         *bool   `json:"due_ok,omitempty"`
	StatusCorrect *bool   `json:"status_correct,omitempty"`
	ErrorType     *string `json:"error_type"`
	Evidence      string  `json:"evidence,omitempty"`
	Reason        string  `json:"reason,omitempty"`
}

type Omission struct {
	Task     string `json:"task"`
	Evidence string `json:"evidence"`
	Reason   string `json:"reason"`
}

type JudgeResult struct {
	Source    string      `json:"source"`
	Items     []JudgeItem `json:"items"`
	Omissions []Omission  `json:"omissions"`
	Verdict   string      `json:"verdict"`
	Summary   string      `json:"summary"`
	Error     string      `json:"error,omitempty"`
	Usage     *Usage      `json:"usage,omitempty"`
}

type Usage struct {
	LatencyMs        int     `json:"latency_ms"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	Cost             float64 `json:"cost"`
}

type Result struct {
	ID               string       `json:"id"`
	RunID            string       `json:"run_id"`
	CaseID           string       `json:"case_id"`
	Repeat           int          `json:"repeat_index"`
	RawOutput        string       `json:"raw_output"`
	Parsed           *ModelOutput `json:"parsed"`
	Rule             RuleResult   `json:"rule"`
	Judge            JudgeResult  `json:"judge"`
	Verdict          string       `json:"verdict"` // pass | fail | error
	HardFails        []Finding    `json:"hard_fails"`
	SoftIssues       []Finding    `json:"soft_issues"`
	Quality          int          `json:"quality"`
	LatencyMs        int          `json:"latency_ms"`
	PromptTokens     int          `json:"prompt_tokens"`
	CompletionTokens int          `json:"completion_tokens"`
	Cost             float64      `json:"cost"`
	Error            string       `json:"error,omitempty"`
	Simulated        bool         `json:"simulated"`
}

type Thresholds struct {
	MinPassRate      float64 `json:"min_pass_rate"`
	MaxHardFailCases int     `json:"max_hard_fail_cases"`
	MaxRegressions   int     `json:"max_regressions"`
}

type GateReason struct {
	Level   string `json:"level"` // block | review | pass
	Message string `json:"message"`
}

type GateDecision struct {
	ID             string       `json:"id"`
	BaselineRunID  string       `json:"baseline_run_id"`
	CandidateRunID string       `json:"candidate_run_id"`
	Decision       string       `json:"decision"` // pass | review | block
	Thresholds     Thresholds   `json:"thresholds"`
	Reasons        []GateReason `json:"reasons"`
	CreatedAt      string       `json:"created_at"`
}

type Data struct {
	Prompts    []Prompt       `json:"prompts"`
	Datasets   []Dataset      `json:"datasets"`
	Cases      []Case         `json:"cases"`
	Runs       []Run          `json:"runs"`
	Results    []Result       `json:"results"`
	Thresholds Thresholds     `json:"thresholds"`
	Gates      []GateDecision `json:"gates"`
}

var DefaultThresholds = Thresholds{
	MinPassRate:      0.85,
	MaxHardFailCases: 1,
	MaxRegressions:   0,
}
