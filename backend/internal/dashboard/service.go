package dashboard

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

// Overview assembles the whole dashboard. Panels are independent, so a panel
// that cannot be computed is not allowed to sink the rest of the page: only a
// failure of the headline KPIs is fatal.
func (s *Service) Overview(ctx context.Context) (Overview, error) {
	kpis, err := s.repo.KPIs(ctx)
	if err != nil {
		return Overview{}, err
	}
	out := Overview{
		KPIs:             kpis,
		HeadcountByDept:  []Bucket{},
		HeadcountByGrade: []Bucket{},
		CompetencyGaps:   []CompetencyGap{},
		ScoreBuckets:     []Bucket{},
		Campaigns:        []CampaignStat{},
		HiringTrend:      []TrendPoint{},
	}
	if v, err := s.repo.HeadcountByDepartment(ctx); err == nil {
		out.HeadcountByDept = v
	}
	if v, err := s.repo.HeadcountByGrade(ctx); err == nil {
		out.HeadcountByGrade = v
	}
	if v, err := s.repo.CompetencyGaps(ctx); err == nil {
		out.CompetencyGaps = v
	}
	if v, err := s.repo.ScoreDistribution(ctx); err == nil {
		out.ScoreBuckets = v
	}
	if v, err := s.repo.CampaignProgress(ctx); err == nil {
		out.Campaigns = v
	}
	if v, err := s.repo.HiringTrend(ctx); err == nil {
		out.HiringTrend = v
	}
	return out, nil
}
