package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// stubCancelRepo is a minimal KnowledgeRepository that only supports the two
// calls CancelKnowledgeParse makes: load-by-id and column update. Rows live in
// an in-memory map keyed by id.
type stubCancelRepo struct {
	interfaces.KnowledgeRepository
	rows map[string]*types.Knowledge
}

func (r *stubCancelRepo) GetKnowledgeByID(_ context.Context, _ uint64, id string) (*types.Knowledge, error) {
	if k, ok := r.rows[id]; ok {
		return k, nil
	}
	return nil, nil
}

func (r *stubCancelRepo) UpdateKnowledgeColumns(_ context.Context, id string, values map[string]interface{}) error {
	k, ok := r.rows[id]
	if !ok {
		return nil
	}
	if v, ok := values["parse_status"].(string); ok {
		k.ParseStatus = v
	}
	return nil
}

func seedCancelRow(r *stubCancelRepo, id, status string) {
	r.rows[id] = &types.Knowledge{ID: id, KnowledgeBaseID: "kb-1", TenantID: 1, ParseStatus: status}
}

// TestBatchCancelKnowledgeParse_ClassifiesByStatus verifies that cancellable
// rows (in-flight, or already-cancelled as an idempotent no-op) are reported as
// cancelled, while rows in a terminal state that cannot be stopped are reported
// as skipped — without erroring the whole batch.
func TestBatchCancelKnowledgeParse_ClassifiesByStatus(t *testing.T) {
	repo := &stubCancelRepo{rows: map[string]*types.Knowledge{}}
	seedCancelRow(repo, "k-run", types.ParseStatusProcessing) // cancellable
	seedCancelRow(repo, "k-done", types.ParseStatusCompleted) // skipped (terminal, cannot stop)
	seedCancelRow(repo, "k-cxl", types.ParseStatusCancelled)  // cancelled (idempotent no-op success)

	svc := &knowledgeService{repo: repo}
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(1))

	cancelled, skipped, err := svc.BatchCancelKnowledgeParse(ctx, []string{"k-run", "k-done", "k-cxl"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(cancelled) != 2 {
		t.Fatalf("cancelled = %v, want [k-run k-cxl]", cancelled)
	}
	if len(skipped) != 1 || skipped[0] != "k-done" {
		t.Fatalf("skipped = %v, want [k-done]", skipped)
	}
	if repo.rows["k-run"].ParseStatus != types.ParseStatusCancelled {
		t.Fatalf("k-run status = %q, want cancelled", repo.rows["k-run"].ParseStatus)
	}
}

// TestBatchCancelKnowledgeParse_MissingRowSkipped verifies a not-found id is
// skipped rather than failing the batch.
func TestBatchCancelKnowledgeParse_MissingRowSkipped(t *testing.T) {
	repo := &stubCancelRepo{rows: map[string]*types.Knowledge{}}
	seedCancelRow(repo, "k-run", types.ParseStatusPending)

	svc := &knowledgeService{repo: repo}
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(1))

	cancelled, skipped, err := svc.BatchCancelKnowledgeParse(ctx, []string{"k-run", "k-missing"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(cancelled) != 1 || cancelled[0] != "k-run" {
		t.Fatalf("cancelled = %v, want [k-run]", cancelled)
	}
	if len(skipped) != 1 || skipped[0] != "k-missing" {
		t.Fatalf("skipped = %v, want [k-missing]", skipped)
	}
}
