package sqlite

import (
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/hypertf/nahcloud/domain"
)

// faultSchemaStatements is intentionally isolated so API integration can move
// these ordered declarations into the shared versioned migration ledger.
func faultSchemaStatements() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS fault_rules (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 1 CHECK(enabled IN (0,1)),
			priority INTEGER NOT NULL DEFAULT 0 CHECK(priority BETWEEN 0 AND 1000),
			operation TEXT,
			route TEXT,
			method TEXT,
			after_matches INTEGER NOT NULL DEFAULT 0 CHECK(after_matches >= 0),
			every_nth INTEGER NOT NULL DEFAULT 1 CHECK(every_nth >= 1),
			failure_percent INTEGER NOT NULL DEFAULT 100 CHECK(failure_percent BETWEEN 1 AND 100),
			seed INTEGER NOT NULL DEFAULT 0,
			max_triggers INTEGER CHECK(max_triggers IS NULL OR max_triggers >= 1),
			status_code INTEGER CHECK(status_code IS NULL OR status_code IN (400,408,409,423,429,500,502,503,504)),
			delay_ms INTEGER NOT NULL DEFAULT 0 CHECK(delay_ms BETWEEN 0 AND 2000),
			match_count INTEGER NOT NULL DEFAULT 0 CHECK(match_count >= 0),
			trigger_count INTEGER NOT NULL DEFAULT 0 CHECK(trigger_count >= 0),
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY (org_id) REFERENCES organizations(id) ON DELETE CASCADE,
			CHECK(operation IS NOT NULL OR route IS NOT NULL OR method IS NOT NULL),
			CHECK(status_code IS NOT NULL OR delay_ms > 0)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_fault_rules_eligible ON fault_rules(org_id, enabled, priority, id)`,
	}
}

type FaultRepository struct{ db *DB }

func NewFaultRepository(db *DB) *FaultRepository { return &FaultRepository{db: db} }

func (r *FaultRepository) Create(rule *domain.FaultRule) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM fault_rules WHERE org_id=?`, rule.OrgID).Scan(&count); err != nil {
		return err
	}
	if count >= domain.MaxFaultRules {
		return domain.ErrFaultRuleLimit
	}
	now := time.Now().UTC()
	rule.CreatedAt, rule.UpdatedAt = now, now
	_, err = tx.Exec(`INSERT INTO fault_rules
		(id,org_id,name,enabled,priority,operation,route,method,after_matches,every_nth,failure_percent,seed,max_triggers,status_code,delay_ms,match_count,trigger_count,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, rule.ID, rule.OrgID, rule.Name, rule.Enabled, rule.Priority, rule.Operation, rule.Route, rule.Method,
		rule.AfterMatches, rule.EveryNth, rule.FailurePercent, rule.Seed, rule.MaxTriggers, rule.StatusCode, rule.DelayMS, 0, 0, now, now)
	if err != nil {
		return fmt.Errorf("create fault rule: %w", err)
	}
	return tx.Commit()
}

const faultRuleColumns = `id,org_id,name,enabled,priority,operation,route,method,after_matches,every_nth,failure_percent,seed,max_triggers,status_code,delay_ms,match_count,trigger_count,created_at,updated_at`

type rowScanner interface{ Scan(...any) error }

func scanFaultRule(row rowScanner) (*domain.FaultRule, error) {
	rule := &domain.FaultRule{}
	var operation, route, method sql.NullString
	var maxTriggers sql.NullInt64
	var statusCode sql.NullInt64
	err := row.Scan(&rule.ID, &rule.OrgID, &rule.Name, &rule.Enabled, &rule.Priority, &operation, &route, &method,
		&rule.AfterMatches, &rule.EveryNth, &rule.FailurePercent, &rule.Seed, &maxTriggers, &statusCode, &rule.DelayMS,
		&rule.MatchCount, &rule.TriggerCount, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if operation.Valid {
		rule.Operation = &operation.String
	}
	if route.Valid {
		rule.Route = &route.String
	}
	if method.Valid {
		rule.Method = &method.String
	}
	if maxTriggers.Valid {
		value := uint64(maxTriggers.Int64)
		rule.MaxTriggers = &value
	}
	if statusCode.Valid {
		value := int(statusCode.Int64)
		rule.StatusCode = &value
	}
	return rule, nil
}

func (r *FaultRepository) List(orgID string) ([]*domain.FaultRule, error) {
	rows, err := r.db.Query(`SELECT `+faultRuleColumns+` FROM fault_rules WHERE org_id=? ORDER BY priority,id`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	rules := make([]*domain.FaultRule, 0)
	for rows.Next() {
		rule, err := scanFaultRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *FaultRepository) Get(orgID, id string) (*domain.FaultRule, error) {
	rule, err := scanFaultRule(r.db.QueryRow(`SELECT `+faultRuleColumns+` FROM fault_rules WHERE org_id=? AND id=?`, orgID, id))
	if err == sql.ErrNoRows {
		return nil, domain.NotFoundError("fault rule", id)
	}
	return rule, err
}

func (r *FaultRepository) Update(rule *domain.FaultRule) error {
	rule.UpdatedAt = time.Now().UTC()
	result, err := r.db.Exec(`UPDATE fault_rules SET name=?,enabled=?,priority=?,operation=?,route=?,method=?,after_matches=?,every_nth=?,failure_percent=?,seed=?,max_triggers=?,status_code=?,delay_ms=?,updated_at=? WHERE org_id=? AND id=?`,
		rule.Name, rule.Enabled, rule.Priority, rule.Operation, rule.Route, rule.Method, rule.AfterMatches, rule.EveryNth, rule.FailurePercent,
		rule.Seed, rule.MaxTriggers, rule.StatusCode, rule.DelayMS, rule.UpdatedAt, rule.OrgID, rule.ID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return domain.NotFoundError("fault rule", rule.ID)
	}
	return nil
}

func (r *FaultRepository) Delete(orgID, id string) error {
	result, err := r.db.Exec(`DELETE FROM fault_rules WHERE org_id=? AND id=?`, orgID, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return domain.NotFoundError("fault rule", id)
	}
	return nil
}

func (r *FaultRepository) Reset(orgID, id string) (*domain.FaultRule, error) {
	result, err := r.db.Exec(`UPDATE fault_rules SET match_count=0,trigger_count=0,updated_at=? WHERE org_id=? AND id=?`, time.Now().UTC(), orgID, id)
	if err != nil {
		return nil, err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return nil, domain.NotFoundError("fault rule", id)
	}
	return r.Get(orgID, id)
}

func (r *FaultRepository) ResetAll(orgID string) error {
	_, err := r.db.Exec(`UPDATE fault_rules SET match_count=0,trigger_count=0,updated_at=? WHERE org_id=?`, time.Now().UTC(), orgID)
	return err
}

func (r *FaultRepository) Evaluate(orgID, operation, route, method string) (*domain.FaultDecision, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rule, err := scanFaultRule(tx.QueryRow(`SELECT `+faultRuleColumns+` FROM fault_rules
		WHERE org_id=? AND enabled=1 AND (max_triggers IS NULL OR trigger_count < max_triggers)
		AND (operation IS NULL OR operation=?) AND (route IS NULL OR route=?) AND (method IS NULL OR method=?)
		ORDER BY priority,id LIMIT 1`, orgID, operation, route, method))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var matchCount uint64
	if err := tx.QueryRow(`UPDATE fault_rules SET match_count=match_count+1 WHERE org_id=? AND id=? RETURNING match_count`, orgID, rule.ID).Scan(&matchCount); err != nil {
		return nil, err
	}
	trigger := matchCount > rule.AfterMatches && (matchCount-rule.AfterMatches)%rule.EveryNth == 0 &&
		deterministicPercent(orgID, rule.ID, rule.Seed, matchCount) < uint64(rule.FailurePercent)
	if !trigger {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if _, err := tx.Exec(`UPDATE fault_rules SET trigger_count=trigger_count+1 WHERE org_id=? AND id=?`, orgID, rule.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &domain.FaultDecision{RuleID: rule.ID, MatchCount: matchCount, StatusCode: rule.StatusCode, DelayMS: rule.DelayMS}, nil
}

func deterministicPercent(orgID, ruleID string, seed int64, matchCount uint64) uint64 {
	// Canonical hash encoding: length-prefixed UTF-8 strings followed by signed
	// seed bits and match count as big-endian 64-bit integers. Selection uses
	// the first eight SHA-256 bytes as an unsigned big-endian integer.
	h := sha256.New()
	writeHashString(h, orgID)
	writeHashString(h, ruleID)
	var number [8]byte
	binary.BigEndian.PutUint64(number[:], uint64(seed))
	h.Write(number[:])
	binary.BigEndian.PutUint64(number[:], matchCount)
	h.Write(number[:])
	return binary.BigEndian.Uint64(h.Sum(nil)[:8]) % 100
}

type hashWriter interface{ Write([]byte) (int, error) }

func writeHashString(w hashWriter, value string) {
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(value)))
	w.Write(length[:])
	w.Write([]byte(value))
}
