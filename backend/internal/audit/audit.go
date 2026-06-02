package audit

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Writer struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

func NewWriter(pool *pgxpool.Pool, log *slog.Logger) *Writer {
	return &Writer{pool: pool, log: log}
}

func (w *Writer) Login(ctx context.Context, userID uuid.UUID, ip, userAgent string) {
	w.write(ctx, &userID, "LOGIN", "users", &userID, nil, nil, ip, userAgent)
}

func (w *Writer) LoginFailed(ctx context.Context, username, ip, userAgent string) {
	payload, _ := json.Marshal(map[string]string{"username": username})
	w.write(ctx, nil, "LOGIN_FAILED", "users", nil, nil, payload, ip, userAgent)
}

func (w *Writer) Record(ctx context.Context, actorID *uuid.UUID, action, entityType string, entityID *uuid.UUID, before, after any, deptScope *uuid.UUID, ip, userAgent string) {
	var beforeJSON, afterJSON []byte
	if before != nil {
		beforeJSON, _ = json.Marshal(before)
	}
	if after != nil {
		afterJSON, _ = json.Marshal(after)
	}
	w.writeWithScope(ctx, actorID, action, entityType, entityID, beforeJSON, afterJSON, deptScope, ip, userAgent)
}

func (w *Writer) write(ctx context.Context, actorID *uuid.UUID, action, entityType string, entityID *uuid.UUID, before, after []byte, ip, userAgent string) {
	w.writeWithScope(ctx, actorID, action, entityType, entityID, before, after, nil, ip, userAgent)
}

func (w *Writer) writeWithScope(ctx context.Context, actorID *uuid.UUID, action, entityType string, entityID *uuid.UUID, before, after []byte, deptScope *uuid.UUID, ip, userAgent string) {
	_, err := w.pool.Exec(ctx, `
		INSERT INTO audit_logs (actor_id, action, entity_type, entity_id, before_data, after_data, department_scope_id, ip, user_agent)
		VALUES ($1, $2::audit_action, $3, $4, $5, $6, $7, NULLIF($8, '')::inet, NULLIF($9, ''))`,
		actorID, action, entityType, entityID, before, after, deptScope, ip, userAgent)
	if err != nil {
		w.log.Error("audit write failed",
			slog.String("action", action),
			slog.String("entity_type", entityType),
			slog.String("err", err.Error()))
	}
}
