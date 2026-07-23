package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/models/embedding"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

type memoryProcessingCache struct {
	mu      sync.Mutex
	entries map[string][]byte
}

func (c *memoryProcessingCache) cacheKey(tenantID uint64, cacheType, key string) string {
	return fmt.Sprintf("%d:%s:%s", tenantID, cacheType, key)
}

func (c *memoryProcessingCache) Get(
	_ context.Context, tenantID uint64, cacheType, key string,
) ([]byte, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	payload, ok := c.entries[c.cacheKey(tenantID, cacheType, key)]
	return append([]byte(nil), payload...), ok, nil
}

func (c *memoryProcessingCache) Put(
	_ context.Context, tenantID uint64, cacheType, key string, payload []byte,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[string][]byte)
	}
	c.entries[c.cacheKey(tenantID, cacheType, key)] = append([]byte(nil), payload...)
	return nil
}

type countingEmbedder struct {
	batchCalls int
	batches    [][]string
}

func (e *countingEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	return []float32{float32(len(text))}, nil
}

func (e *countingEmbedder) BatchEmbed(_ context.Context, texts []string) ([][]float32, error) {
	e.batchCalls++
	e.batches = append(e.batches, append([]string(nil), texts...))
	out := make([][]float32, len(texts))
	for i, text := range texts {
		out[i] = []float32{float32(len(text))}
	}
	return out, nil
}

func (e *countingEmbedder) BatchEmbedWithPool(
	ctx context.Context, _ embedding.Embedder, texts []string,
) ([][]float32, error) {
	return e.BatchEmbed(ctx, texts)
}

func (e *countingEmbedder) GetModelName() string { return "embedding-model" }
func (e *countingEmbedder) GetDimensions() int   { return 1 }
func (e *countingEmbedder) GetModelID() string   { return "embedding-id" }

func TestCachedEmbedderBatchPartialHitsAndDeduplicates(t *testing.T) {
	cache := &memoryProcessingCache{}
	inner := &countingEmbedder{}
	cached := newCachedEmbedder(inner, cache, 7)

	got, err := cached.BatchEmbed(context.Background(), []string{"alpha", "beta", "alpha"})
	require.NoError(t, err)
	require.Equal(t, [][]float32{{5}, {4}, {5}}, got)
	require.Equal(t, 1, inner.batchCalls)
	require.Equal(t, []string{"alpha", "beta"}, inner.batches[0])

	got, err = cached.BatchEmbed(context.Background(), []string{"beta", "alpha"})
	require.NoError(t, err)
	require.Equal(t, [][]float32{{4}, {5}}, got)
	require.Equal(t, 1, inner.batchCalls)
}

type failingProcessingCache struct{}

func (failingProcessingCache) Get(
	context.Context, uint64, string, string,
) ([]byte, bool, error) {
	return nil, false, errors.New("cache unavailable")
}

func (failingProcessingCache) Put(
	context.Context, uint64, string, string, []byte,
) error {
	return errors.New("cache unavailable")
}

func TestCachedEmbedderFailsOpenWhenCacheIsUnavailable(t *testing.T) {
	inner := &countingEmbedder{}
	cached := newCachedEmbedder(inner, failingProcessingCache{}, 7)

	got, err := cached.BatchEmbed(context.Background(), []string{"alpha", "beta"})
	require.NoError(t, err)
	require.Equal(t, [][]float32{{5}, {4}}, got)
	require.Equal(t, 1, inner.batchCalls)
}

type countingChat struct {
	calls int
}

func (c *countingChat) Chat(
	_ context.Context, messages []chat.Message, _ *chat.ChatOptions,
) (*types.ChatResponse, error) {
	c.calls++
	return &types.ChatResponse{Content: fmt.Sprintf("response-%d:%s", c.calls, messages[0].Content)}, nil
}

func (c *countingChat) ChatStream(
	context.Context, []chat.Message, *chat.ChatOptions,
) (<-chan types.StreamResponse, error) {
	return nil, nil
}

func (c *countingChat) GetModelName() string { return "chat-model" }
func (c *countingChat) GetModelID() string   { return "chat-id" }

func TestCachedChatIgnoresVolatileImageURLButKeepsSemanticText(t *testing.T) {
	cache := &memoryProcessingCache{}
	inner := &countingChat{}
	cached := newCachedChat(inner, cache, 9, cacheTypeWikiChat)
	ctx := context.Background()

	first, err := cached.Chat(ctx, []chat.Message{{
		Role:    "user",
		Content: `![page](resource://first)<image_ocr>same text</image_ocr>`,
	}}, &chat.ChatOptions{Temperature: 0.1})
	require.NoError(t, err)

	second, err := cached.Chat(ctx, []chat.Message{{
		Role:    "user",
		Content: `![page](resource://second)<image_ocr>same text</image_ocr>`,
	}}, &chat.ChatOptions{Temperature: 0.1})
	require.NoError(t, err)
	require.Equal(t, first.Content, second.Content)
	require.Equal(t, 1, inner.calls)

	_, err = cached.Chat(ctx, []chat.Message{{
		Role:    "user",
		Content: `![page](resource://second)<image_ocr>changed text</image_ocr>`,
	}}, &chat.ChatOptions{Temperature: 0.1})
	require.NoError(t, err)
	require.Equal(t, 2, inner.calls)
}

type countingVLM struct {
	calls int
}

func (v *countingVLM) Predict(_ context.Context, _ [][]byte, prompt string) (string, error) {
	v.calls++
	return fmt.Sprintf("%s-%d", prompt, v.calls), nil
}

func (v *countingVLM) GetModelName() string { return "vlm-model" }
func (v *countingVLM) GetModelID() string   { return "vlm-id" }

func TestCachedVLMKeysImageBytesAndPrompt(t *testing.T) {
	cache := &memoryProcessingCache{}
	inner := &countingVLM{}
	cached := newCachedVLM(inner, cache, 11)
	ctx := context.Background()

	first, err := cached.Predict(ctx, [][]byte{[]byte("image-a")}, "ocr-v1")
	require.NoError(t, err)
	second, err := cached.Predict(ctx, [][]byte{[]byte("image-a")}, "ocr-v1")
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, 1, inner.calls)

	_, err = cached.Predict(ctx, [][]byte{[]byte("image-a")}, "ocr-v2")
	require.NoError(t, err)
	_, err = cached.Predict(ctx, [][]byte{[]byte("image-b")}, "ocr-v1")
	require.NoError(t, err)
	require.Equal(t, 3, inner.calls)
}

func TestCachedVLMFailsOpenOnCorruptPayload(t *testing.T) {
	cache := &memoryProcessingCache{entries: make(map[string][]byte)}
	inner := &countingVLM{}
	cached := newCachedVLM(inner, cache, 11)
	ctx := context.Background()

	image := [][]byte{[]byte("image-a")}
	imageHash := sha256.Sum256(image[0])
	key, err := contentCacheKey(struct {
		ModelID    string
		ModelName  string
		ImageHash  []string
		PromptText string
	}{
		ModelID:    inner.GetModelID(),
		ModelName:  inner.GetModelName(),
		ImageHash:  []string{hex.EncodeToString(imageHash[:])},
		PromptText: "ocr-v1",
	})
	require.NoError(t, err)
	cache.entries[cache.cacheKey(11, cacheTypeVLM, key)] = []byte("{not-json")

	got, err := cached.Predict(ctx, image, "ocr-v1")
	require.NoError(t, err)
	require.Equal(t, "ocr-v1-1", got)
	require.Equal(t, 1, inner.calls)
}

func TestStableChunkIDNormalizesImageStorageURL(t *testing.T) {
	first := stableChunkID("knowledge", types.ChunkTypeText, 3, "before ![page](resource://first) after")
	second := stableChunkID("knowledge", types.ChunkTypeText, 3, "before ![page](resource://second) after")
	require.Equal(t, first, second)

	changed := stableChunkID("knowledge", types.ChunkTypeText, 3, "different text")
	require.NotEqual(t, first, changed)
}
