package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// Build-progress handler test. The handler validates KB read access via
// validateAndGetKnowledgeBase, then returns the service's aggregate snapshot.

type stubProgressKB struct {
	interfaces.KnowledgeBaseService
}

func (s *stubProgressKB) GetKnowledgeBaseByID(_ context.Context, id string) (*types.KnowledgeBase, error) {
	return &types.KnowledgeBase{ID: id, TenantID: 1, Type: "document"}, nil
}

func (s *stubProgressKB) GetKnowledgeBuildProgress(_ context.Context, _ string, _ uint64) (*types.KnowledgeBuildProgress, error) {
	return &types.KnowledgeBuildProgress{Total: 10, Completed: 6, Processing: 2, Pending: 1, Failed: 1}, nil
}

func newBuildProgressRouter(kb interfaces.KnowledgeBaseService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		c.Set(types.TenantIDContextKey.String(), uint64(1))
		c.Set(types.UserIDContextKey.String(), "u-test")
		c.Next()
	})
	h := &KnowledgeBaseHandler{service: kb}
	r.GET("/knowledge-bases/:id/build-progress", h.GetKnowledgeBuildProgress)
	return r
}

func TestGetKnowledgeBuildProgress_ReturnsSnapshot(t *testing.T) {
	router := newBuildProgressRouter(&stubProgressKB{})
	req := httptest.NewRequest(http.MethodGet, "/knowledge-bases/kb-1/build-progress", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool                         `json:"success"`
		Data    types.KnowledgeBuildProgress `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v; body=%s", err, w.Body.String())
	}
	if !resp.Success || resp.Data.Total != 10 || resp.Data.Completed != 6 || resp.Data.Processing != 2 {
		t.Fatalf("unexpected data: %+v", resp.Data)
	}
}
