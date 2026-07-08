package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// Batch cancel-parse handler tests. The handler must dedup ids, enforce the
// 200-per-batch cap, verify KB access + same-KB membership, and return
// cancelled/skipped counts from the service.

type stubBatchCancelKG struct {
	interfaces.KnowledgeService
}

func (s *stubBatchCancelKG) GetKnowledgeBatch(_ context.Context, _ uint64, ids []string) ([]*types.Knowledge, error) {
	out := make([]*types.Knowledge, 0, len(ids))
	for _, id := range ids {
		out = append(out, &types.Knowledge{ID: id, KnowledgeBaseID: "kb-1", TenantID: 1})
	}
	return out, nil
}

func (s *stubBatchCancelKG) BatchCancelKnowledgeParse(_ context.Context, ids []string) ([]string, []string, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}
	// First cancelled, rest skipped — lets the test assert both counts.
	return ids[:1], ids[1:], nil
}

type stubBatchCancelKB struct {
	interfaces.KnowledgeBaseService
}

func (s *stubBatchCancelKB) GetKnowledgeBaseByID(_ context.Context, id string) (*types.KnowledgeBase, error) {
	return &types.KnowledgeBase{ID: id, TenantID: 1, Type: "document"}, nil
}

func newBatchCancelRouter(kg interfaces.KnowledgeService, kb interfaces.KnowledgeBaseService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		// KB TenantID matches this tenant, so validateKnowledgeBaseAccessWithKBID
		// resolves OrgRoleAdmin and the Editor/Admin gate passes.
		c.Set(types.TenantIDContextKey.String(), uint64(1))
		c.Set(types.UserIDContextKey.String(), "u-test")
		c.Next()
	})
	h := &KnowledgeHandler{kgService: kg, kbService: kb}
	r.POST("/batch-cancel-parse", h.BatchCancelKnowledgeParse)
	return r
}

func TestBatchCancelKnowledgeParse_ReturnsCounts(t *testing.T) {
	router := newBatchCancelRouter(&stubBatchCancelKG{}, &stubBatchCancelKB{})
	body, _ := json.Marshal(map[string]any{"kb_id": "kb-1", "ids": []string{"k1", "k2"}})
	req := httptest.NewRequest(http.MethodPost, "/batch-cancel-parse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			CancelledCount int `json:"cancelled_count"`
			SkippedCount   int `json:"skipped_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v; body=%s", err, w.Body.String())
	}
	if !resp.Success || resp.Data.CancelledCount != 1 || resp.Data.SkippedCount != 1 {
		t.Fatalf("unexpected resp: %+v", resp)
	}
}

func TestBatchCancelKnowledgeParse_EmptyIDs(t *testing.T) {
	router := newBatchCancelRouter(&stubBatchCancelKG{}, &stubBatchCancelKB{})
	body, _ := json.Marshal(map[string]any{"kb_id": "kb-1", "ids": []string{"  ", ""}})
	req := httptest.NewRequest(http.MethodPost, "/batch-cancel-parse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	_ = strings.TrimSpace("")
}
