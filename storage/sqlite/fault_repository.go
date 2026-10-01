package sqlite

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/hypertf/nahcloud/domain"
)

type FaultRepository struct{ db *DB }

func NewFaultRepository(db *DB) *FaultRepository { return &FaultRepository{db: db} }

func (r *FaultRepository) Put(orgID string, req domain.PutFaultScenarioRequest) (*domain.FaultScenario, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO fault_scenarios (org_id,name,enabled,seed,updated_at) VALUES (?,?,?,?,CURRENT_TIMESTAMP)
		ON CONFLICT(org_id) DO UPDATE SET name=excluded.name,enabled=excluded.enabled,seed=excluded.seed,updated_at=CURRENT_TIMESTAMP`, orgID, req.Name, req.Enabled, req.Seed); err != nil {
		return nil, fmt.Errorf("put fault scenario: %w", err)
	}
	if _, err = tx.Exec(`DELETE FROM fault_rules WHERE org_id=?`, orgID); err != nil {
		return nil, err
	}
	for i, rule := range req.Rules {
		if _, err = tx.Exec(`INSERT INTO fault_rules (org_id,id,position,operation,route,status,latency_ms,every_n) VALUES (?,?,?,?,?,?,?,?)`,
			orgID, rule.ID, i, rule.Operation, rule.Route, rule.Status, rule.LatencyMS, rule.EveryN); err != nil {
			return nil, fmt.Errorf("put fault rule: %w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(orgID)
}

func (r *FaultRepository) Get(orgID string) (*domain.FaultScenario, error) {
	s := &domain.FaultScenario{OrgID: orgID, Rules: []domain.FaultRule{}}
	var enabled int
	if err := r.db.QueryRow(`SELECT name,enabled,seed,updated_at FROM fault_scenarios WHERE org_id=?`, orgID).Scan(&s.Name, &enabled, &s.Seed, &s.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NotFoundError("fault scenario", orgID)
		}
		return nil, err
	}
	s.Enabled = enabled != 0
	rows, err := r.db.Query(`SELECT id,operation,route,status,latency_ms,every_n,call_count,injected_count FROM fault_rules WHERE org_id=? ORDER BY position`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var rule domain.FaultRule
		if err := rows.Scan(&rule.ID, &rule.Operation, &rule.Route, &rule.Status, &rule.LatencyMS, &rule.EveryN, &rule.CallCount, &rule.InjectedCount); err != nil {
			return nil, err
		}
		s.Rules = append(s.Rules, rule)
	}
	return s, rows.Err()
}

func (r *FaultRepository) Reset(orgID string) (*domain.FaultScenario, error) {
	result, err := r.db.Exec(`UPDATE fault_rules SET call_count=0,injected_count=0 WHERE org_id=?`, orgID)
	if err != nil {
		return nil, err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		if _, err := r.Get(orgID); err != nil {
			return nil, err
		}
	}
	return r.Get(orgID)
}

func (r *FaultRepository) Evaluate(orgID, operation, route string) (*domain.FaultDecision, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var seed int64
	var enabled int
	if err := tx.QueryRow(`SELECT enabled,seed FROM fault_scenarios WHERE org_id=?`, orgID).Scan(&enabled, &seed); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if enabled == 0 {
		return nil, nil
	}
	rows, err := tx.Query(`SELECT id,route,status,latency_ms,every_n FROM fault_rules WHERE org_id=? AND operation=? ORDER BY position`, orgID, strings.ToUpper(operation))
	if err != nil {
		return nil, err
	}
	var rule domain.FaultRule
	found := false
	for rows.Next() {
		if err := rows.Scan(&rule.ID, &rule.Route, &rule.Status, &rule.LatencyMS, &rule.EveryN); err != nil {
			rows.Close()
			return nil, err
		}
		if routeMatches(rule.Route, route) {
			found = true
			break
		}
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	var call uint64
	if err := tx.QueryRow(`UPDATE fault_rules SET call_count=call_count+1 WHERE org_id=? AND id=? RETURNING call_count`, orgID, rule.ID).Scan(&call); err != nil {
		return nil, err
	}
	if deterministicSlot(uint64(seed), rule.ID, call, uint64(rule.EveryN)) != 0 {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if _, err := tx.Exec(`UPDATE fault_rules SET injected_count=injected_count+1 WHERE org_id=? AND id=?`, orgID, rule.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &domain.FaultDecision{RuleID: rule.ID, Call: call, Status: rule.Status, LatencyMS: rule.LatencyMS}, nil
}

func routeMatches(pattern, route string) bool {
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(route, strings.TrimSuffix(pattern, "*"))
	}
	return pattern == route
}

func deterministicSlot(seed uint64, ruleID string, call, modulo uint64) uint64 {
	h := fnv.New64a()
	var value [8]byte
	binary.LittleEndian.PutUint64(value[:], seed)
	h.Write(value[:])
	h.Write([]byte(ruleID))
	return (h.Sum64()%modulo + (call-1)%modulo) % modulo
}
