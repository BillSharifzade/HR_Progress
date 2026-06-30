package competency

import (
	"context"
	"encoding/json"
	"errors"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrMemberNotInPeriod = errors.New("user is not a member of any group in this period")

// rawAssessorScore is one assessor mark used by the grouping engine.
type rawAssessorScore struct {
	EmployeeID   uuid.UUID
	CompetencyID uuid.UUID
	Score        float64
}

// FormGroups (re)builds learning groups for a campaign from assessor scores
// only (FR-AS13). Employees are ranked ascending by their mean assessor score
// and chunked into groups of groupSize; each group gets its competency
// analytics (strength + 2–3 dev zones). Existing groups are replaced.
func (r *Repository) FormGroups(ctx context.Context, periodID uuid.UUID, groupSize int, actorID uuid.UUID) error {
	if groupSize < 1 {
		groupSize = 12
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Assessees of this campaign.
	assRows, err := tx.Query(ctx, `SELECT user_id FROM assessment_assessees WHERE period_id = $1`, periodID)
	if err != nil {
		return err
	}
	assessees := map[uuid.UUID]struct{}{}
	for assRows.Next() {
		var id uuid.UUID
		if err := assRows.Scan(&id); err != nil {
			assRows.Close()
			return err
		}
		assessees[id] = struct{}{}
	}
	assRows.Close()

	// Raw assessor scores for those assessees.
	raw, err := fetchRawAssessorScores(ctx, tx, periodID)
	if err != nil {
		return err
	}

	// Per-employee mean over all assessor scores across all competencies.
	type acc struct {
		sum float64
		n   int
	}
	emp := map[uuid.UUID]*acc{}
	for _, s := range raw {
		if _, ok := assessees[s.EmployeeID]; !ok {
			continue
		}
		a := emp[s.EmployeeID]
		if a == nil {
			a = &acc{}
			emp[s.EmployeeID] = a
		}
		a.sum += s.Score
		a.n++
	}

	type ranked struct {
		userID uuid.UUID
		avg    float64
	}
	list := make([]ranked, 0, len(emp))
	for id, a := range emp {
		if a.n == 0 {
			continue
		}
		list = append(list, ranked{userID: id, avg: a.sum / float64(a.n)})
	}
	// Ascending by avg (ties broken by id for determinism).
	sort.Slice(list, func(i, j int) bool {
		if list[i].avg != list[j].avg {
			return list[i].avg < list[j].avg
		}
		return list[i].userID.String() < list[j].userID.String()
	})

	// Wipe previous groups for this campaign.
	if _, err := tx.Exec(ctx, `DELETE FROM learning_groups WHERE period_id = $1`, periodID); err != nil {
		return err
	}

	groupNo := 0
	for start := 0; start < len(list); start += groupSize {
		end := start + groupSize
		if end > len(list) {
			end = len(list)
		}
		chunk := list[start:end]
		groupNo++

		members := make([]uuid.UUID, 0, len(chunk))
		scoreMin, scoreMax := chunk[0].avg, chunk[0].avg
		for _, m := range chunk {
			members = append(members, m.userID)
			if m.avg < scoreMin {
				scoreMin = m.avg
			}
			if m.avg > scoreMax {
				scoreMax = m.avg
			}
		}

		strengthComp, strengthScore, zones := analyzeGroup(raw, members)

		var groupID uuid.UUID
		var strengthArg any
		if strengthComp != uuid.Nil {
			strengthArg = strengthComp
		}
		err := tx.QueryRow(ctx, `
			INSERT INTO learning_groups (period_id, group_no, score_min, score_max, strength_competency_id, strength_score)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			periodID, groupNo, scoreMin, scoreMax, strengthArg, nullFloat(strengthScore)).Scan(&groupID)
		if err != nil {
			return err
		}
		for pos, m := range chunk {
			if _, err := tx.Exec(ctx, `
				INSERT INTO learning_group_members (group_id, period_id, user_id, avg_score, position)
				VALUES ($1, $2, $3, $4, $5)`, groupID, periodID, m.userID, m.avg, start+pos+1); err != nil {
				return err
			}
		}
		for rank, z := range zones {
			if _, err := tx.Exec(ctx, `
				INSERT INTO learning_group_dev_zones (group_id, competency_id, avg_score, rank)
				VALUES ($1, $2, $3, $4)`, groupID, z.compID, z.avg, rank+1); err != nil {
				return err
			}
		}
	}

	if err := journalTx(ctx, tx, periodID, nil, "form_groups", map[string]any{
		"group_size": groupSize, "groups": groupNo, "ranked": len(list),
	}, actorID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func fetchRawAssessorScores(ctx context.Context, q pgx.Tx, periodID uuid.UUID) ([]rawAssessorScore, error) {
	rows, err := q.Query(ctx, `
		SELECT s.employee_id, s.competency_id, s.score
		FROM assessment_scores s
		JOIN assessment_assessees aa ON aa.period_id = s.period_id AND aa.user_id = s.employee_id
		WHERE s.period_id = $1 AND s.assessor_role = 'ASSESSOR' AND s.score IS NOT NULL`, periodID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]rawAssessorScore, 0)
	for rows.Next() {
		var s rawAssessorScore
		if err := rows.Scan(&s.EmployeeID, &s.CompetencyID, &s.Score); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

type zone struct {
	compID uuid.UUID
	avg    float64
}

// analyzeGroup computes the per-competency average over the given members and
// returns the strongest competency and the 2–3 weakest (dev zones).
func analyzeGroup(raw []rawAssessorScore, members []uuid.UUID) (uuid.UUID, float64, []zone) {
	memberSet := map[uuid.UUID]struct{}{}
	for _, m := range members {
		memberSet[m] = struct{}{}
	}
	type acc struct {
		sum float64
		n   int
	}
	byComp := map[uuid.UUID]*acc{}
	for _, s := range raw {
		if _, ok := memberSet[s.EmployeeID]; !ok {
			continue
		}
		a := byComp[s.CompetencyID]
		if a == nil {
			a = &acc{}
			byComp[s.CompetencyID] = a
		}
		a.sum += s.Score
		a.n++
	}
	if len(byComp) == 0 {
		return uuid.Nil, 0, nil
	}
	avgs := make([]zone, 0, len(byComp))
	for id, a := range byComp {
		avgs = append(avgs, zone{compID: id, avg: a.sum / float64(a.n)})
	}
	// Strongest = highest avg.
	sort.Slice(avgs, func(i, j int) bool {
		if avgs[i].avg != avgs[j].avg {
			return avgs[i].avg > avgs[j].avg
		}
		return avgs[i].compID.String() < avgs[j].compID.String()
	})
	strength := avgs[0]
	// Weakest 2–3 = lowest avg.
	asc := make([]zone, len(avgs))
	copy(asc, avgs)
	sort.Slice(asc, func(i, j int) bool {
		if asc[i].avg != asc[j].avg {
			return asc[i].avg < asc[j].avg
		}
		return asc[i].compID.String() < asc[j].compID.String()
	})
	n := 3
	if len(asc) < n {
		n = len(asc)
	}
	return strength.compID, strength.avg, asc[:n]
}

func nullFloat(f float64) any {
	if f == 0 {
		return nil
	}
	return f
}

func journalTx(ctx context.Context, tx pgx.Tx, periodID uuid.UUID, groupID *uuid.UUID, action string, detail map[string]any, actorID uuid.UUID) error {
	var raw []byte
	if detail != nil {
		raw, _ = json.Marshal(detail)
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO learning_group_journal (period_id, group_id, action, detail, actor_id)
		VALUES ($1, $2, $3, $4, $5)`, periodID, groupID, action, raw, actorID)
	return err
}

// ── Reads ────────────────────────────────────────────────────────────────────

func (r *Repository) ListGroups(ctx context.Context, periodID uuid.UUID) ([]LearningGroup, error) {
	rows, err := r.db.Query(ctx, `
		SELECT g.id, g.period_id, g.group_no, g.score_min, g.score_max,
		       g.strength_competency_id, c.name, g.strength_score, g.confirmed, g.formed_at
		FROM learning_groups g
		LEFT JOIN competencies c ON c.id = g.strength_competency_id
		WHERE g.period_id = $1
		ORDER BY g.group_no`, periodID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := make([]LearningGroup, 0)
	index := map[uuid.UUID]int{}
	for rows.Next() {
		var g LearningGroup
		var strengthName *string
		if err := rows.Scan(&g.ID, &g.PeriodID, &g.GroupNo, &g.ScoreMin, &g.ScoreMax,
			&g.StrengthCompetencyID, &strengthName, &g.StrengthScore, &g.Confirmed, &g.FormedAt); err != nil {
			return nil, err
		}
		if strengthName != nil {
			g.StrengthName = *strengthName
		}
		g.Members = []GroupMember{}
		g.DevZones = []DevZone{}
		index[g.ID] = len(groups)
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return groups, nil
	}

	mRows, err := r.db.Query(ctx, `
		SELECT m.id, m.group_id, m.user_id, u.full_name, m.avg_score, m.position
		FROM learning_group_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.period_id = $1
		ORDER BY m.position`, periodID)
	if err != nil {
		return nil, err
	}
	for mRows.Next() {
		var m GroupMember
		if err := mRows.Scan(&m.ID, &m.GroupID, &m.UserID, &m.FullName, &m.AvgScore, &m.Position); err != nil {
			mRows.Close()
			return nil, err
		}
		if i, ok := index[m.GroupID]; ok {
			groups[i].Members = append(groups[i].Members, m)
		}
	}
	mRows.Close()

	zRows, err := r.db.Query(ctx, `
		SELECT z.group_id, z.competency_id, c.name, z.avg_score, z.rank
		FROM learning_group_dev_zones z
		JOIN learning_groups g ON g.id = z.group_id
		JOIN competencies c ON c.id = z.competency_id
		WHERE g.period_id = $1
		ORDER BY z.rank`, periodID)
	if err != nil {
		return nil, err
	}
	defer zRows.Close()
	for zRows.Next() {
		var groupID uuid.UUID
		var z DevZone
		if err := zRows.Scan(&groupID, &z.CompetencyID, &z.CompetencyName, &z.AvgScore, &z.Rank); err != nil {
			return nil, err
		}
		if i, ok := index[groupID]; ok {
			groups[i].DevZones = append(groups[i].DevZones, z)
		}
	}
	return groups, zRows.Err()
}

func (r *Repository) ListGroupJournal(ctx context.Context, periodID uuid.UUID) ([]GroupJournalEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, group_id, action, detail::text, actor_id, at
		FROM learning_group_journal
		WHERE period_id = $1
		ORDER BY at DESC
		LIMIT 500`, periodID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]GroupJournalEntry, 0)
	for rows.Next() {
		var e GroupJournalEntry
		if err := rows.Scan(&e.ID, &e.GroupID, &e.Action, &e.Detail, &e.ActorID, &e.At); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ── Manual edits (FR-AS13.11) ────────────────────────────────────────────────

// MoveMember moves an employee to another group and recomputes analytics for
// both affected groups. The move is journaled.
func (r *Repository) MoveMember(ctx context.Context, periodID, userID, toGroupID uuid.UUID, actorID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var fromGroupID uuid.UUID
	var avg float64
	err = tx.QueryRow(ctx,
		`SELECT group_id, avg_score FROM learning_group_members WHERE period_id = $1 AND user_id = $2`,
		periodID, userID).Scan(&fromGroupID, &avg)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrMemberNotInPeriod
	}
	if err != nil {
		return err
	}
	if fromGroupID == toGroupID {
		return nil
	}
	// Verify target group belongs to the same campaign.
	var ok bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM learning_groups WHERE id = $1 AND period_id = $2)`, toGroupID, periodID).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}

	if _, err := tx.Exec(ctx,
		`UPDATE learning_group_members SET group_id = $1 WHERE period_id = $2 AND user_id = $3`,
		toGroupID, periodID, userID); err != nil {
		return err
	}
	raw, err := fetchRawAssessorScores(ctx, tx, periodID)
	if err != nil {
		return err
	}
	if err := recomputeGroup(ctx, tx, fromGroupID, raw); err != nil {
		return err
	}
	if err := recomputeGroup(ctx, tx, toGroupID, raw); err != nil {
		return err
	}
	if err := journalTx(ctx, tx, periodID, &toGroupID, "move_member", map[string]any{
		"user_id": userID.String(), "from": fromGroupID.String(), "to": toGroupID.String(),
	}, actorID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func recomputeGroup(ctx context.Context, tx pgx.Tx, groupID uuid.UUID, raw []rawAssessorScore) error {
	rows, err := tx.Query(ctx, `SELECT user_id, avg_score FROM learning_group_members WHERE group_id = $1`, groupID)
	if err != nil {
		return err
	}
	members := []uuid.UUID{}
	var min, max float64
	first := true
	for rows.Next() {
		var id uuid.UUID
		var avg float64
		if err := rows.Scan(&id, &avg); err != nil {
			rows.Close()
			return err
		}
		members = append(members, id)
		if first || avg < min {
			min = avg
		}
		if first || avg > max {
			max = avg
		}
		first = false
	}
	rows.Close()

	if len(members) == 0 {
		// Empty group: clear analytics.
		_, err := tx.Exec(ctx, `DELETE FROM learning_group_dev_zones WHERE group_id = $1`, groupID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`UPDATE learning_groups SET score_min = NULL, score_max = NULL, strength_competency_id = NULL, strength_score = NULL WHERE id = $1`,
			groupID)
		return err
	}

	strengthComp, strengthScore, zones := analyzeGroup(raw, members)
	var strengthArg any
	if strengthComp != uuid.Nil {
		strengthArg = strengthComp
	}
	if _, err := tx.Exec(ctx,
		`UPDATE learning_groups SET score_min = $2, score_max = $3, strength_competency_id = $4, strength_score = $5 WHERE id = $1`,
		groupID, min, max, strengthArg, nullFloat(strengthScore)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM learning_group_dev_zones WHERE group_id = $1`, groupID); err != nil {
		return err
	}
	for rank, z := range zones {
		if _, err := tx.Exec(ctx,
			`INSERT INTO learning_group_dev_zones (group_id, competency_id, avg_score, rank) VALUES ($1,$2,$3,$4)`,
			groupID, z.compID, z.avg, rank+1); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) SetGroupSize(ctx context.Context, periodID uuid.UUID, size int) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE assessment_periods SET group_size = $2, updated_at = now() WHERE id = $1`, periodID, size)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ConfirmGroups(ctx context.Context, periodID uuid.UUID, actorID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE learning_groups SET confirmed = true WHERE period_id = $1`, periodID); err != nil {
		return err
	}
	if err := journalTx(ctx, tx, periodID, nil, "confirm_groups", nil, actorID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ── Service layer ────────────────────────────────────────────────────────────

func (s *Service) ListGroups(ctx context.Context, periodID uuid.UUID) ([]LearningGroup, error) {
	return s.repo.ListGroups(ctx, periodID)
}

func (s *Service) ListGroupJournal(ctx context.Context, periodID uuid.UUID) ([]GroupJournalEntry, error) {
	return s.repo.ListGroupJournal(ctx, periodID)
}

// RegenerateGroups re-runs grouping with the current (or a new) group size.
func (s *Service) RegenerateGroups(ctx context.Context, periodID uuid.UUID, newSize *int, actorID uuid.UUID) error {
	p, err := s.repo.GetPeriod(ctx, periodID)
	if err != nil {
		return ErrNotFound
	}
	if p.Status != StatusConfirmed && p.Status != StatusAdminReview {
		return errors.New("groups can only be (re)formed while the campaign is in review or confirmed")
	}
	size := p.GroupSize
	if newSize != nil && *newSize > 0 {
		size = *newSize
		if err := s.repo.SetGroupSize(ctx, periodID, size); err != nil {
			return err
		}
	}
	return s.repo.FormGroups(ctx, periodID, size, actorID)
}

func (s *Service) MoveMember(ctx context.Context, periodID, userID, toGroupID, actorID uuid.UUID) error {
	return s.repo.MoveMember(ctx, periodID, userID, toGroupID, actorID)
}

func (s *Service) ConfirmGroups(ctx context.Context, periodID, actorID uuid.UUID) error {
	return s.repo.ConfirmGroups(ctx, periodID, actorID)
}
