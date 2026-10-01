package sqlite

import (
	"fmt"
	"sync"
	"testing"

	"github.com/hypertf/nahcloud/domain"
	"github.com/stretchr/testify/require"
)

func ptr[T any](value T) *T { return &value }

func newFaultRepositoryTest(t *testing.T) (*DB, *FaultRepository) {
	t.Helper()
	db, err := NewDB(t.TempDir() + "/fault.db")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	for _, id := range []string{"one", "two"} {
		require.NoError(t, NewOrganizationRepository(db).Create(&domain.Organization{ID: id, Slug: id, Name: id}))
	}
	return db, NewFaultRepository(db)
}

func testFaultRule(id, org string, priority int) *domain.FaultRule {
	return &domain.FaultRule{ID: id, OrgID: org, Enabled: true, Priority: priority, Method: ptr("GET"), EveryNth: 1, FailurePercent: 100, StatusCode: ptr(503)}
}

func TestFaultSelectionOrderingSchedulingAndExhaustion(t *testing.T) {
	_, repo := newFaultRepositoryTest(t)
	first := testFaultRule("00000000000000000000000000000001", "one", 1)
	first.Operation, first.Route = ptr("projects.get"), ptr("/v1/projects/{project}")
	first.AfterMatches, first.EveryNth, first.MaxTriggers = 1, 2, ptr(uint64(1))
	second := testFaultRule("00000000000000000000000000000002", "one", 2)
	second.Operation, second.Route = ptr("projects.get"), ptr("/v1/projects/{project}")
	require.NoError(t, repo.Create(second))
	require.NoError(t, repo.Create(first))

	// Near misses on each ANDed dimension select nothing.
	for _, input := range [][3]string{{"projects.list", "/v1/projects/{project}", "GET"}, {"projects.get", "/v1/projects", "GET"}, {"projects.get", "/v1/projects/{project}", "POST"}} {
		decision, err := repo.Evaluate("one", input[0], input[1], input[2])
		require.NoError(t, err)
		require.Nil(t, decision)
	}

	// The winning rule counts and shadows lower priority even when it does not trigger.
	decision, err := repo.Evaluate("one", "projects.get", "/v1/projects/{project}", "GET")
	require.NoError(t, err)
	require.Nil(t, decision)
	decision, err = repo.Evaluate("one", "projects.get", "/v1/projects/{project}", "GET")
	require.NoError(t, err)
	require.Nil(t, decision)
	decision, err = repo.Evaluate("one", "projects.get", "/v1/projects/{project}", "GET")
	require.NoError(t, err)
	require.Equal(t, first.ID, decision.RuleID)
	storedFirst, err := repo.Get("one", first.ID)
	require.NoError(t, err)
	require.Equal(t, uint64(3), storedFirst.MatchCount)
	require.Equal(t, uint64(1), storedFirst.TriggerCount)
	storedSecond, err := repo.Get("one", second.ID)
	require.NoError(t, err)
	require.Zero(t, storedSecond.MatchCount)

	// Exhaustion skips the first rule and lets the next candidate win.
	decision, err = repo.Evaluate("one", "projects.get", "/v1/projects/{project}", "GET")
	require.NoError(t, err)
	require.Equal(t, second.ID, decision.RuleID)
	require.Equal(t, uint64(1), decision.MatchCount)
}

func TestFaultSelectionUsesIDTieBreakAndIgnoresDisabled(t *testing.T) {
	_, repo := newFaultRepositoryTest(t)
	disabled := testFaultRule("00000000000000000000000000000000", "one", 0)
	disabled.Enabled = false
	first := testFaultRule("00000000000000000000000000000001", "one", 0)
	second := testFaultRule("00000000000000000000000000000002", "one", 0)
	require.NoError(t, repo.Create(second))
	require.NoError(t, repo.Create(disabled))
	require.NoError(t, repo.Create(first))

	decision, err := repo.Evaluate("one", "", "", "GET")
	require.NoError(t, err)
	require.Equal(t, first.ID, decision.RuleID)
	stored, err := repo.Get("one", disabled.ID)
	require.NoError(t, err)
	require.Zero(t, stored.MatchCount)
	stored, err = repo.Get("one", second.ID)
	require.NoError(t, err)
	require.Zero(t, stored.MatchCount)
}

func TestFaultCountersAreAtomicAndTenantScoped(t *testing.T) {
	_, repo := newFaultRepositoryTest(t)
	rule := testFaultRule("10000000000000000000000000000000", "one", 0)
	rule.MaxTriggers = ptr(uint64(5))
	require.NoError(t, repo.Create(rule))
	const calls = 40
	var wg sync.WaitGroup
	triggered := make(chan bool, calls)
	errs := make(chan error, calls)
	for i := 0; i < calls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			decision, err := repo.Evaluate("one", "", "", "GET")
			errs <- err
			triggered <- decision != nil
		}()
	}
	wg.Wait()
	close(triggered)
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	count := 0
	for value := range triggered {
		if value {
			count++
		}
	}
	require.Equal(t, 5, count)
	stored, err := repo.Get("one", rule.ID)
	require.NoError(t, err)
	require.Equal(t, uint64(5), stored.MatchCount)
	require.Equal(t, uint64(5), stored.TriggerCount)
	decision, err := repo.Evaluate("two", "", "", "GET")
	require.NoError(t, err)
	require.Nil(t, decision)
	_, err = repo.Get("two", rule.ID)
	require.True(t, domain.IsNotFound(err))
}

func TestFaultDeterminismResetsAndPersistence(t *testing.T) {
	_, repo := newFaultRepositoryTest(t)
	rule := testFaultRule("20000000000000000000000000000000", "one", 0)
	rule.FailurePercent, rule.Seed = 37, -9
	require.NoError(t, repo.Create(rule))
	require.Equal(t, uint64(97), deterministicPercent("one", rule.ID, -9, 1))
	require.Equal(t, uint64(99), deterministicPercent("one", rule.ID, -8, 1))
	sequence := func() []bool {
		values := make([]bool, 12)
		for i := range values {
			decision, err := repo.Evaluate("one", "", "", "GET")
			require.NoError(t, err)
			values[i] = decision != nil
		}
		return values
	}
	first := sequence()
	reset, err := repo.Reset("one", rule.ID)
	require.NoError(t, err)
	require.Zero(t, reset.MatchCount)
	require.Equal(t, first, sequence())
	require.NoError(t, repo.ResetAll("one"))
}

func TestFaultRuleLimitIsAtomic(t *testing.T) {
	_, repo := newFaultRepositoryTest(t)
	for i := 0; i < domain.MaxFaultRules; i++ {
		rule := testFaultRule(fmt.Sprintf("%032x", i+1), "one", 0)
		require.NoError(t, repo.Create(rule))
	}
	err := repo.Create(testFaultRule("ffffffffffffffffffffffffffffffff", "one", 0))
	require.ErrorIs(t, err, domain.ErrFaultRuleLimit)
}

func TestFaultRulePersistsAcrossDatabaseReopen(t *testing.T) {
	path := t.TempDir() + "/fault.db"
	db, err := NewDB(path)
	require.NoError(t, err)
	require.NoError(t, NewOrganizationRepository(db).Create(&domain.Organization{ID: "org", Slug: "org", Name: "org"}))
	repo := NewFaultRepository(db)
	rule := testFaultRule("30000000000000000000000000000000", "org", 0)
	require.NoError(t, repo.Create(rule))
	_, err = repo.Evaluate("org", "", "", "GET")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	db, err = NewDB(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	stored, err := NewFaultRepository(db).Get("org", rule.ID)
	require.NoError(t, err)
	require.Equal(t, uint64(1), stored.MatchCount)
	require.Equal(t, uint64(1), stored.TriggerCount)
}
