package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/models/embedding"
	"github.com/Tencent/WeKnora/internal/models/vlm"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

const (
	cacheTypeParse     = "parse"
	cacheTypeEmbedding = "embedding"
	cacheTypeVLM       = "vlm"
	cacheTypeSummary   = "chat_summary"
	cacheTypeQuestion  = "chat_question"
	cacheTypeTable     = "chat_table"
	cacheTypeWikiMap   = "wiki_map"
	cacheTypeWikiChat  = "chat_wiki_map"
	cacheTypeGraph     = "chat_graph"
)

var (
	cacheMarkdownImagePattern = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	cacheImageURLPattern      = regexp.MustCompile(`(?i)<image\s+url="[^"]*">`)
	cacheImageOriginalPattern = regexp.MustCompile(`(?is)<image_original>.*?</image_original>`)
)

func contentCacheKey(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// normalizeCacheText removes storage-generated image identities while keeping
// all semantic text, including OCR and captions. Object URLs are deliberately
// excluded because image bytes are keyed independently at the VLM layer.
func normalizeCacheText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = cacheMarkdownImagePattern.ReplaceAllString(text, "![$1]([image])")
	text = cacheImageURLPattern.ReplaceAllString(text, `<image url="[image]">`)
	text = cacheImageOriginalPattern.ReplaceAllString(text, "<image_original>[image]</image_original>")
	return strings.TrimSpace(text)
}

func stableChunkID(knowledgeID string, chunkType types.ChunkType, sequence int, content string) string {
	name := strings.Join([]string{
		knowledgeID,
		chunkType,
		strconv.Itoa(sequence),
		normalizeCacheText(content),
	}, "\x00")
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(name)).String()
}

func cacheGetJSON(
	ctx context.Context,
	repo interfaces.ProcessingCacheRepository,
	tenantID uint64,
	cacheType, key string,
	target any,
) bool {
	if repo == nil || tenantID == 0 || key == "" {
		return false
	}
	payload, found, err := repo.Get(ctx, tenantID, cacheType, key)
	if err != nil {
		logger.Warnf(ctx, "processing cache get failed (type=%s): %v", cacheType, err)
		return false
	}
	if !found {
		return false
	}
	if err := json.Unmarshal(payload, target); err != nil {
		logger.Warnf(ctx, "processing cache decode failed (type=%s): %v", cacheType, err)
		return false
	}
	return true
}

func cachePutJSON(
	ctx context.Context,
	repo interfaces.ProcessingCacheRepository,
	tenantID uint64,
	cacheType, key string,
	value any,
) {
	if repo == nil || tenantID == 0 || key == "" {
		return
	}
	payload, err := json.Marshal(value)
	if err != nil {
		logger.Warnf(ctx, "processing cache encode failed (type=%s): %v", cacheType, err)
		return
	}
	if err := repo.Put(ctx, tenantID, cacheType, key, payload); err != nil {
		logger.Warnf(ctx, "processing cache put failed (type=%s): %v", cacheType, err)
	}
}

type cachedChat struct {
	inner     chat.Chat
	repo      interfaces.ProcessingCacheRepository
	tenantID  uint64
	cacheType string
}

func newCachedChat(
	inner chat.Chat,
	repo interfaces.ProcessingCacheRepository,
	tenantID uint64,
	cacheType string,
) chat.Chat {
	if inner == nil || repo == nil || tenantID == 0 {
		return inner
	}
	return &cachedChat{inner: inner, repo: repo, tenantID: tenantID, cacheType: cacheType}
}

func (c *cachedChat) GetModelName() string { return c.inner.GetModelName() }
func (c *cachedChat) GetModelID() string   { return c.inner.GetModelID() }

func (c *cachedChat) Chat(
	ctx context.Context,
	messages []chat.Message,
	opts *chat.ChatOptions,
) (*types.ChatResponse, error) {
	canonicalMessages := make([]chat.Message, len(messages))
	copy(canonicalMessages, messages)
	for i := range canonicalMessages {
		canonicalMessages[i].Content = normalizeCacheText(canonicalMessages[i].Content)
	}
	key, err := contentCacheKey(struct {
		ModelID   string
		ModelName string
		Messages  []chat.Message
		Options   *chat.ChatOptions
	}{
		ModelID:   c.inner.GetModelID(),
		ModelName: c.inner.GetModelName(),
		Messages:  canonicalMessages,
		Options:   opts,
	})
	if err == nil {
		var cached types.ChatResponse
		if cacheGetJSON(ctx, c.repo, c.tenantID, c.cacheType, key, &cached) {
			cached.Usage = types.TokenUsage{}
			logger.Infof(ctx, "processing cache hit (type=%s key=%s)", c.cacheType, key[:12])
			return &cached, nil
		}
	}

	response, callErr := c.inner.Chat(ctx, messages, opts)
	if callErr == nil && response != nil && err == nil {
		cachePutJSON(ctx, c.repo, c.tenantID, c.cacheType, key, response)
	}
	return response, callErr
}

func (c *cachedChat) ChatStream(
	ctx context.Context,
	messages []chat.Message,
	opts *chat.ChatOptions,
) (<-chan types.StreamResponse, error) {
	return c.inner.ChatStream(ctx, messages, opts)
}

type cachedVLM struct {
	inner    vlm.VLM
	repo     interfaces.ProcessingCacheRepository
	tenantID uint64
}

func newCachedVLM(
	inner vlm.VLM,
	repo interfaces.ProcessingCacheRepository,
	tenantID uint64,
) vlm.VLM {
	if inner == nil || repo == nil || tenantID == 0 {
		return inner
	}
	return &cachedVLM{inner: inner, repo: repo, tenantID: tenantID}
}

func (c *cachedVLM) GetModelName() string { return c.inner.GetModelName() }
func (c *cachedVLM) GetModelID() string   { return c.inner.GetModelID() }

func (c *cachedVLM) Predict(ctx context.Context, images [][]byte, prompt string) (string, error) {
	imageHashes := make([]string, len(images))
	for i, image := range images {
		sum := sha256.Sum256(image)
		imageHashes[i] = hex.EncodeToString(sum[:])
	}
	key, err := contentCacheKey(struct {
		ModelID    string
		ModelName  string
		ImageHash  []string
		PromptText string
	}{
		ModelID:    c.inner.GetModelID(),
		ModelName:  c.inner.GetModelName(),
		ImageHash:  imageHashes,
		PromptText: prompt,
	})
	if err == nil {
		var cached string
		if cacheGetJSON(ctx, c.repo, c.tenantID, cacheTypeVLM, key, &cached) {
			logger.Infof(ctx, "processing cache hit (type=%s key=%s)", cacheTypeVLM, key[:12])
			return cached, nil
		}
	}

	result, callErr := c.inner.Predict(ctx, images, prompt)
	if callErr == nil && err == nil {
		cachePutJSON(ctx, c.repo, c.tenantID, cacheTypeVLM, key, result)
	}
	return result, callErr
}

type cachedEmbedder struct {
	inner    embedding.Embedder
	repo     interfaces.ProcessingCacheRepository
	tenantID uint64
}

func newCachedEmbedder(
	inner embedding.Embedder,
	repo interfaces.ProcessingCacheRepository,
	tenantID uint64,
) embedding.Embedder {
	if inner == nil || repo == nil || tenantID == 0 {
		return inner
	}
	return &cachedEmbedder{inner: inner, repo: repo, tenantID: tenantID}
}

func (c *cachedEmbedder) GetModelName() string { return c.inner.GetModelName() }
func (c *cachedEmbedder) GetModelID() string   { return c.inner.GetModelID() }
func (c *cachedEmbedder) GetDimensions() int   { return c.inner.GetDimensions() }

func (c *cachedEmbedder) embeddingKey(text string) (string, error) {
	return contentCacheKey(struct {
		ModelID    string
		ModelName  string
		Dimensions int
		Text       string
	}{
		ModelID:    c.inner.GetModelID(),
		ModelName:  c.inner.GetModelName(),
		Dimensions: c.inner.GetDimensions(),
		Text:       normalizeCacheText(text),
	})
}

func (c *cachedEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	results, err := c.embedBatch(ctx, []string{text}, func(missing []string) ([][]float32, error) {
		vector, embedErr := c.inner.Embed(ctx, missing[0])
		return [][]float32{vector}, embedErr
	})
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

func (c *cachedEmbedder) BatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	return c.embedBatch(ctx, texts, func(missing []string) ([][]float32, error) {
		return c.inner.BatchEmbed(ctx, missing)
	})
}

func (c *cachedEmbedder) BatchEmbedWithPool(
	ctx context.Context,
	_ embedding.Embedder,
	texts []string,
) ([][]float32, error) {
	return c.embedBatch(ctx, texts, func(missing []string) ([][]float32, error) {
		return c.inner.BatchEmbedWithPool(ctx, c.inner, missing)
	})
}

func (c *cachedEmbedder) embedBatch(
	ctx context.Context,
	texts []string,
	fetch func([]string) ([][]float32, error),
) ([][]float32, error) {
	results := make([][]float32, len(texts))
	keys := make([]string, len(texts))
	missingTexts := make([]string, 0, len(texts))
	missingIndexes := make(map[string][]int)

	for i, text := range texts {
		key, err := c.embeddingKey(text)
		if err != nil {
			key = "uncacheable:" + strconv.Itoa(i)
			keys[i] = key
			missingTexts = append(missingTexts, text)
			missingIndexes[key] = []int{i}
			continue
		}
		keys[i] = key
		var vector []float32
		if cacheGetJSON(ctx, c.repo, c.tenantID, cacheTypeEmbedding, key, &vector) {
			results[i] = vector
			continue
		}
		if _, exists := missingIndexes[key]; !exists {
			missingTexts = append(missingTexts, text)
		}
		missingIndexes[key] = append(missingIndexes[key], i)
	}

	if len(missingTexts) == 0 {
		logger.Infof(ctx, "processing cache hit (type=%s count=%d)", cacheTypeEmbedding, len(texts))
		return results, nil
	}

	fetched, err := fetch(missingTexts)
	if err != nil {
		return nil, err
	}
	if len(fetched) != len(missingTexts) {
		return nil, &cacheResultCountError{expected: len(missingTexts), actual: len(fetched)}
	}

	fetchedIndex := 0
	seen := make(map[string]bool)
	for i := range texts {
		key := keys[i]
		if results[i] != nil || seen[key] {
			continue
		}
		seen[key] = true
		vector := fetched[fetchedIndex]
		fetchedIndex++
		for _, index := range missingIndexes[key] {
			results[index] = append([]float32(nil), vector...)
		}
		if key != "" {
			cachePutJSON(ctx, c.repo, c.tenantID, cacheTypeEmbedding, key, vector)
		}
	}
	return results, nil
}

type cacheResultCountError struct {
	expected int
	actual   int
}

func (e *cacheResultCountError) Error() string {
	return fmt.Sprintf(
		"embedding provider returned %d results for %d inputs",
		e.actual,
		e.expected,
	)
}
