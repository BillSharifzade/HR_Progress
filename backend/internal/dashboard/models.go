package dashboard

// Overview is the whole dashboard payload. It is deliberately one response:
// the page renders as a unit, and a single round trip keeps every panel
// consistent with the same snapshot of the data.
type Overview struct {
	KPIs             KPIs            `json:"kpis"`
	HeadcountByDept  []Bucket        `json:"headcount_by_department"`
	HeadcountByGrade []Bucket        `json:"headcount_by_grade"`
	CompetencyGaps   []CompetencyGap `json:"competency_gaps"`
	ScoreBuckets     []Bucket        `json:"score_distribution"`
	Campaigns        []CampaignStat  `json:"campaign_progress"`
	HiringTrend      []TrendPoint    `json:"hiring_trend"`
}

// KPIs are the headline numbers shown as stat tiles.
type KPIs struct {
	ActiveWorkers   int `json:"active_workers"`
	Departments     int `json:"departments"`
	Competencies    int `json:"competencies"`
	ActiveCampaigns int `json:"active_campaigns"`
	AssessedWorkers int `json:"assessed_workers"`
	// AvgScore is nil until at least one consolidated mark exists.
	AvgScore *float64 `json:"avg_score"`
}

// Bucket is one labelled count — the shape every bar chart consumes.
type Bucket struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

// CompetencyGap contrasts the organisation's average mark for a competency
// with the minimum its requirements demand. Delta < 0 is a development zone.
type CompetencyGap struct {
	CompetencyID string  `json:"competency_id"`
	Name         string  `json:"name"`
	Kind         string  `json:"kind"`
	AvgScore     float64 `json:"avg_score"`
	RequiredMin  float64 `json:"required_min"`
	Delta        float64 `json:"delta"`
	Assessed     int     `json:"assessed"`
}

// CampaignStat is completion progress for one running campaign: how many
// (assessee × criterion) cells have a consolidated mark out of those expected.
type CampaignStat struct {
	PeriodID  string `json:"period_id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Assessees int    `json:"assessees"`
	Criteria  int    `json:"criteria"`
	Done      int    `json:"done"`
	Expected  int    `json:"expected"`
}

// TrendPoint is one month of the hiring trend. Month is "YYYY-MM".
type TrendPoint struct {
	Month string `json:"month"`
	Hired int    `json:"hired"`
	Total int    `json:"total"` // running headcount at end of month
}
