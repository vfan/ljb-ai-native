package report

// Result 是学业报告的稳定输出结构。
// Agent、HTTP Handler 和前端都可以复用它，而不必解析自然语言。
type Result struct {
	Summary     string   `json:"summary"`
	Strengths   []string `json:"strengths"`
	WeakCourses []string `json:"weak_courses"`
	NextActions []string `json:"next_actions"`
}
