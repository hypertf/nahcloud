package sqlite

import (
	"fmt"
	"sync"
	"testing"

	"github.com/hypertf/nahcloud/domain"
	"github.com/stretchr/testify/require"
)

func TestFaultRepositoryDeterministicBoundariesAndReset(t *testing.T) {
	db, err := NewDB(t.TempDir() + "/fault.db")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	orgs := NewOrganizationRepository(db)
	for _, id := range []string{"one", "two"} {
		require.NoError(t, orgs.Create(&domain.Organization{ID: id, Slug: id, Name: id}))
	}
	repo := NewFaultRepository(db)
	req := domain.PutFaultScenarioRequest{Enabled: true, Seed: 91, Rules: []domain.FaultRule{{
		ID: "read", Operation: "GET", Route: "/v1/projects/*", Status: 503, EveryN: 3,
	}}}
	_, err = repo.Put("one", req)
	require.NoError(t, err)

	sequence := func() []bool {
		got := make([]bool, 8)
		for i := range got {
			decision, err := repo.Evaluate("one", "GET", "/v1/projects/example")
			require.NoError(t, err)
			got[i] = decision != nil
		}
		return got
	}
	first := sequence()
	injected := 0
	for _, value := range first {
		if value {
			injected++
		}
	}
	require.Contains(t, []int{2, 3}, injected, "an eight-call window must contain one injection per three calls")
	_, err = repo.Reset("one")
	require.NoError(t, err)
	require.Equal(t, first, sequence(), "reset with the same seed must replay the sequence")

	decision, err := repo.Evaluate("one", "GET", "/v1/project")
	require.NoError(t, err)
	require.Nil(t, decision, "prefix matching must not match a shorter path")
	scenario, err := repo.Get("one")
	require.NoError(t, err)
	require.Equal(t, uint64(8), scenario.Rules[0].CallCount)

	decision, err = repo.Evaluate("two", "GET", "/v1/projects/example")
	require.NoError(t, err)
	require.Nil(t, decision, "an organization without a scenario is unaffected")
}

func TestFaultRepositoryCountersAreAtomic(t *testing.T) {
	db, err := NewDB(t.TempDir() + "/fault.db")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, NewOrganizationRepository(db).Create(&domain.Organization{ID: "org", Slug: "org", Name: "org"}))
	repo := NewFaultRepository(db)
	_, err = repo.Put("org", domain.PutFaultScenarioRequest{Enabled: true, Rules: []domain.FaultRule{{
		ID: "all", Operation: "POST", Route: "/v1/projects", Status: 429, EveryN: 1,
	}}})
	require.NoError(t, err)

	const calls = 40
	var wg sync.WaitGroup
	errs := make(chan error, calls)
	for i := 0; i < calls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			decision, err := repo.Evaluate("org", "POST", "/v1/projects")
			if err == nil && (decision == nil || decision.Status != 429) {
				err = fmt.Errorf("missing decision")
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	scenario, err := repo.Get("org")
	require.NoError(t, err)
	require.Equal(t, uint64(calls), scenario.Rules[0].CallCount)
	require.Equal(t, uint64(calls), scenario.Rules[0].InjectedCount)
}

func TestDisabledFaultScenarioDoesNotAdvanceCounters(t *testing.T) {
	db, err := NewDB(t.TempDir() + "/fault.db")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, NewOrganizationRepository(db).Create(&domain.Organization{ID: "org", Slug: "org", Name: "org"}))
	repo := NewFaultRepository(db)
	_, err = repo.Put("org", domain.PutFaultScenarioRequest{Enabled: false, Rules: []domain.FaultRule{{ID: "off", Operation: "GET", Route: "/v1/org", Status: 500, EveryN: 1}}})
	require.NoError(t, err)
	decision, err := repo.Evaluate("org", "GET", "/v1/org")
	require.NoError(t, err)
	require.Nil(t, decision)
	scenario, err := repo.Get("org")
	require.NoError(t, err)
	require.Zero(t, scenario.Rules[0].CallCount)
}

func TestFaultScenarioPersistsAcrossDatabaseReopen(t *testing.T) {
	path := t.TempDir() + "/fault.db"
	db, err := NewDB(path)
	require.NoError(t, err)
	require.NoError(t, NewOrganizationRepository(db).Create(&domain.Organization{ID: "org", Slug: "org", Name: "org"}))
	repo := NewFaultRepository(db)
	_, err = repo.Put("org", domain.PutFaultScenarioRequest{Name: "persistent", Enabled: true, Seed: 42, Rules: []domain.FaultRule{{
		ID: "persist", Operation: "DELETE", Route: "/v1/projects/*", Status: 409, EveryN: 2,
	}}})
	require.NoError(t, err)
	_, err = repo.Evaluate("org", "DELETE", "/v1/projects/demo")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	db, err = NewDB(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	scenario, err := NewFaultRepository(db).Get("org")
	require.NoError(t, err)
	require.Equal(t, "persistent", scenario.Name)
	require.Equal(t, int64(42), scenario.Seed)
	require.Equal(t, uint64(1), scenario.Rules[0].CallCount)
}
