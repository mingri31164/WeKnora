package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// stubProgressRepo implements only the two count methods
// GetKnowledgeBuildProgress needs. byStatus is keyed by the first status in the
// query slice, which is how the service groups its per-bucket queries.
type stubProgressRepo struct {
	interfaces.KnowledgeRepository
	total    int64
	byStatus map[string]int64
}

func (r *stubProgressRepo) CountKnowledgeByKnowledgeBaseID(_ context.Context, _ uint64, _ string) (int64, error) {
	return r.total, nil
}

func (r *stubProgressRepo) CountKnowledgeByStatus(_ context.Context, _ uint64, _ string, statuses []string) (int64, error) {
	if len(statuses) == 0 {
		return 0, nil
	}
	return r.byStatus[statuses[0]], nil
}

func TestGetKnowledgeBuildProgress_Aggregates(t *testing.T) {
	repo := &stubProgressRepo{
		total: 10,
		byStatus: map[string]int64{
			types.ParseStatusCompleted:  6,
			types.ParseStatusProcessing: 2, // processing bucket queries processing first
			types.ParseStatusPending:    1,
			types.ParseStatusFailed:     1,
			types.ParseStatusCancelled:  0,
		},
	}
	svc := &knowledgeBaseService{kgRepo: repo}

	p, err := svc.GetKnowledgeBuildProgress(context.Background(), "kb-1", 1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.Total != 10 || p.Completed != 6 || p.Processing != 2 || p.Pending != 1 || p.Failed != 1 || p.Cancelled != 0 {
		t.Fatalf("unexpected progress: %+v", p)
	}
}

func TestGetKnowledgeBuildProgress_EmptyKB(t *testing.T) {
	repo := &stubProgressRepo{total: 0, byStatus: map[string]int64{}}
	svc := &knowledgeBaseService{kgRepo: repo}

	p, err := svc.GetKnowledgeBuildProgress(context.Background(), "kb-empty", 1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.Total != 0 || p.Completed != 0 {
		t.Fatalf("unexpected progress for empty KB: %+v", p)
	}
}
