package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type wikiCacheKnowledgeService struct {
	interfaces.KnowledgeService
	knowledge *types.Knowledge
}

func (s *wikiCacheKnowledgeService) GetKnowledgeByIDOnly(
	context.Context, string,
) (*types.Knowledge, error) {
	return s.knowledge, nil
}

type wikiCacheChunkRepository struct {
	interfaces.ChunkRepository
	chunks []*types.Chunk
}

func (r *wikiCacheChunkRepository) ListChunksByKnowledgeID(
	context.Context, uint64, string,
) ([]*types.Chunk, error) {
	return r.chunks, nil
}

func (r *wikiCacheChunkRepository) ListChunksByParentIDs(
	context.Context, uint64, []string,
) ([]*types.Chunk, error) {
	return nil, nil
}

type wikiCachePageService struct {
	interfaces.WikiPageService
	slugs []string
}

func (s *wikiCachePageService) ListSlugsBySourceRef(
	context.Context, string, string,
) ([]string, error) {
	return s.slugs, nil
}

func TestWikiMapArtifactCacheSkipsMapLLMAndStillBuildsRetractions(t *testing.T) {
	const (
		tenantID    = uint64(12)
		knowledgeID = "knowledge-1"
		kbID        = "kb-1"
	)
	chunk := &types.Chunk{
		ID:              "chunk-1",
		TenantID:        tenantID,
		KnowledgeID:     knowledgeID,
		KnowledgeBaseID: kbID,
		ChunkIndex:      0,
		ChunkType:       types.ChunkTypeText,
		Content:         "Acme builds a retrieval platform. This text is long enough for wiki extraction.",
	}
	knowledge := &types.Knowledge{
		ID:          knowledgeID,
		TenantID:    tenantID,
		Title:       "Acme document",
		ParseStatus: types.ParseStatusCompleted,
	}
	cache := &memoryProcessingCache{}
	model := &countingChat{}
	batchCtx := &WikiBatchContext{
		ExtractionGranularity: types.WikiExtractionStandard,
		SummaryContentByKnowledgeID: func(context.Context, string) string {
			return "previous contribution"
		},
	}
	lang := types.LanguageLocaleName("en")
	key := buildWikiMapCacheKey(knowledgeID, model, lang, chunk.Content, []*types.Chunk{chunk}, batchCtx)
	cachePutJSON(context.Background(), cache, tenantID, cacheTypeWikiMap, key, wikiMapCacheArtifact{
		Entities: []extractedItem{{
			Name:        "Acme",
			Slug:        "entity/acme",
			Description: "A retrieval platform company",
		}},
		SummaryContent: "SUMMARY: Acme overview\nAcme builds a retrieval platform.",
		Citations: map[string][]string{
			"entity/acme": {"chunk-1"},
		},
		BatchCount: 1,
	})

	svc := &wikiIngestService{
		knowledgeSvc: &wikiCacheKnowledgeService{knowledge: knowledge},
		chunkRepo:    &wikiCacheChunkRepository{chunks: []*types.Chunk{chunk}},
		wikiService:  &wikiCachePageService{slugs: []string{"entity/acme", "summary/knowledge-1"}},
		cacheRepo:    cache,
	}
	result, updates, err := svc.mapOneDocument(
		context.Background(),
		model,
		WikiIngestPayload{TenantID: tenantID, KnowledgeBaseID: kbID},
		WikiPendingOp{Op: WikiOpIngest, KnowledgeID: knowledgeID, Language: "en"},
		batchCtx,
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Zero(t, model.calls)
	require.Equal(t, true, result.MapStats["cache_hit"])

	updateTypes := make(map[string]int)
	for _, update := range updates {
		updateTypes[update.Type]++
	}
	require.Equal(t, 1, updateTypes[types.WikiPageTypeSummary])
	require.Equal(t, 1, updateTypes[types.WikiPageTypeEntity])
	require.Equal(t, 1, updateTypes["retract"])
}
