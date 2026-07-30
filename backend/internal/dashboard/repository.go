package dashboard

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

// activeCampaignStatuses are the statuses that count as "in flight" — a
// campaign someone is still working on. Draft is excluded (not started yet)
// and so is published (finished).
const activeCampaignStatuses = `('assigned', 'in_progress', 'admin_review', 'confirmed')`

func (r *Repository) KPIs(ctx context.Context) (KPIs, error) {
	var k KPIs
	err := r.db.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM users
		    WHERE deleted_at IS NULL AND is_active),
		  (SELECT count(*) FROM departments
		    WHERE deleted_at IS NULL AND is_active),
		  (SELECT count(*) FROM competencies
		    WHERE deleted_at IS NULL AND is_active),
		  (SELECT count(*) FROM assessment_periods
		    WHERE status IN `+activeCampaignStatuses+`),
		  (SELECT count(DISTINCT employee_id) FROM assessment_consolidated),
		  (SELECT AVG(avg_score) FROM assessment_consolidated)`,
	).Scan(&k.ActiveWorkers, &k.Departments, &k.Competencies,
		&k.ActiveCampaigns, &k.AssessedWorkers, &k.AvgScore)
	return k, err
}

func (r *Repository) HeadcountByDepartment(ctx context.Context) ([]Bucket, error) {
	return r.buckets(ctx, `
		SELECT d.name, count(u.id)
		  FROM departments d
		  JOIN users u ON u.department_id = d.id
		   AND u.deleted_at IS NULL AND u.is_active
		 WHERE d.deleted_at IS NULL AND d.is_active
		 GROUP BY d.name
		 HAVING count(u.id) > 0
		 ORDER BY count(u.id) DESC`)
}

func (r *Repository) HeadcountByGrade(ctx context.Context) ([]Bucket, error) {
	// Ordered by the grade ladder, not by size — the axis carries that order.
	return r.buckets(ctx, `
		SELECT g.name, count(u.id)
		  FROM grades g
		  LEFT JOIN users u ON u.grade_id = g.id
		   AND u.deleted_at IS NULL AND u.is_active
		 WHERE g.deleted_at IS NULL AND g.is_active
		 GROUP BY g.name, g.level
		 ORDER BY g.level`)
}

// ScoreDistribution buckets every consolidated mark into five 2-point bands
// across the 0–10 scale. Bands are emitted even when empty so the histogram
// keeps a stable x-axis.
func (r *Repository) ScoreDistribution(ctx context.Context) ([]Bucket, error) {
	return r.buckets(ctx, `
		WITH bands(lo, hi, label) AS (
		  VALUES (0, 2, '0–2'), (2, 4, '2–4'), (4, 6, '4–6'),
		         (6, 8, '6–8'), (8, 10, '8–10')
		)
		SELECT b.label,
		       (SELECT count(*) FROM assessment_consolidated ac
		         WHERE ac.avg_score >= b.lo
		           AND (ac.avg_score < b.hi OR (b.hi = 10 AND ac.avg_score = 10)))
		  FROM bands b
		 ORDER BY b.lo`)
}

func (r *Repository) buckets(ctx context.Context, sql string) ([]Bucket, error) {
	rows, err := r.db.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Bucket, 0)
	for rows.Next() {
		var b Bucket
		if err := rows.Scan(&b.Label, &b.Value); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// CompetencyGaps compares each competency's mean consolidated mark against the
// mean minimum its department/grade requirements demand. Only competencies
// that have both a mark and a configured minimum can produce a gap.
func (r *Repository) CompetencyGaps(ctx context.Context) ([]CompetencyGap, error) {
	rows, err := r.db.Query(ctx, `
		WITH scored AS (
		  SELECT competency_id, AVG(avg_score) AS avg_score, count(*) AS n
		    FROM assessment_consolidated
		   GROUP BY competency_id
		),
		required AS (
		  SELECT competency_id, AVG(required_min::numeric) AS required_min
		    FROM dept_competency_requirements
		   WHERE required_min IS NOT NULL
		   GROUP BY competency_id
		)
		SELECT c.id, c.name, c.kind::text,
		       s.avg_score, r.required_min, s.avg_score - r.required_min, s.n
		  FROM competencies c
		  JOIN scored   s ON s.competency_id = c.id
		  JOIN required r ON r.competency_id = c.id
		 WHERE c.deleted_at IS NULL AND c.is_active
		 ORDER BY s.avg_score - r.required_min`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CompetencyGap, 0)
	for rows.Next() {
		var g CompetencyGap
		if err := rows.Scan(&g.CompetencyID, &g.Name, &g.Kind,
			&g.AvgScore, &g.RequiredMin, &g.Delta, &g.Assessed); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *Repository) CampaignProgress(ctx context.Context) ([]CampaignStat, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.id, p.title, p.status,
		       (SELECT count(*) FROM assessment_assessees   a WHERE a.period_id = p.id),
		       (SELECT count(*) FROM assessment_criteria    c WHERE c.period_id = p.id),
		       (SELECT count(*) FROM assessment_consolidated k WHERE k.period_id = p.id)
		  FROM assessment_periods p
		 WHERE p.status IN `+activeCampaignStatuses+`
		 ORDER BY p.period_start DESC
		 LIMIT 8`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CampaignStat, 0)
	for rows.Next() {
		var c CampaignStat
		if err := rows.Scan(&c.PeriodID, &c.Title, &c.Status,
			&c.Assessees, &c.Criteria, &c.Done); err != nil {
			return nil, err
		}
		c.Expected = c.Assessees * c.Criteria
		out = append(out, c)
	}
	return out, rows.Err()
}

// HiringTrend returns the last 12 months of hires plus the running headcount,
// so the chart can show both the monthly bars and the cumulative curve.
// Workers with no hired_at (not every 1F record carries one) are counted in
// the running total's starting offset, never as a spike in an arbitrary month.
func (r *Repository) HiringTrend(ctx context.Context) ([]TrendPoint, error) {
	rows, err := r.db.Query(ctx, `
		WITH months AS (
		  SELECT date_trunc('month', now()) - (n || ' months')::interval AS m
		    FROM generate_series(11, 0, -1) AS n
		),
		hires AS (
		  SELECT date_trunc('month', hired_at) AS m, count(*) AS n
		    FROM users
		   WHERE deleted_at IS NULL AND hired_at IS NOT NULL
		   GROUP BY 1
		)
		SELECT to_char(months.m, 'YYYY-MM'),
		       COALESCE(h.n, 0),
		       (SELECT count(*) FROM users u
		         WHERE u.deleted_at IS NULL
		           AND (u.hired_at IS NULL OR u.hired_at < months.m + interval '1 month'))
		  FROM months
		  LEFT JOIN hires h ON h.m = months.m
		 ORDER BY months.m`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TrendPoint, 0)
	for rows.Next() {
		var p TrendPoint
		if err := rows.Scan(&p.Month, &p.Hired, &p.Total); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
