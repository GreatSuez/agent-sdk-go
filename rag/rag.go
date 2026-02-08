// Package rag provides retrieval-augmented generation support.
//
// RAG allows agents to augment LLM context with relevant documents
// retrieved from a vector store. This package defines the core
// interfaces and provides an in-memory implementation.
package rag

import (
	"context"
	"math"
	"sort"
	"sync"
)

// Document represents a chunk of text with metadata and its embedding vector.
type Document struct {
	ID        string         `json:"id"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Embedding []float64      `json:"embedding,omitempty"`
}

// SearchResult pairs a document with its similarity score.
type SearchResult struct {
	Document Document `json:"document"`
	Score    float64  `json:"score"`
}

// Embedder converts text into a vector embedding.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float64, error)
}

// VectorStore persists and searches document embeddings.
type VectorStore interface {
	// Add stores documents with their embeddings.
	Add(ctx context.Context, docs []Document) error
	// Search finds the top-k most similar documents to the query vector.
	Search(ctx context.Context, queryVec []float64, topK int) ([]SearchResult, error)
	// Delete removes documents by ID.
	Delete(ctx context.Context, ids []string) error
	// Count returns the number of stored documents.
	Count() int
}

// Retriever combines embedding and search into a single query interface.
type Retriever interface {
	// Retrieve finds relevant documents for a text query.
	Retrieve(ctx context.Context, query string, topK int) ([]SearchResult, error)
}

// SimpleRetriever combines an Embedder and VectorStore.
type SimpleRetriever struct {
	Embedder Embedder
	Store    VectorStore
}

func (r *SimpleRetriever) Retrieve(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	vec, err := r.Embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}
	return r.Store.Search(ctx, vec, topK)
}

// MemoryStore is an in-memory vector store using cosine similarity.
type MemoryStore struct {
	mu   sync.RWMutex
	docs []Document
}

// NewMemoryStore creates an empty in-memory vector store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (m *MemoryStore) Add(_ context.Context, docs []Document) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.docs = append(m.docs, docs...)
	return nil
}

func (m *MemoryStore) Search(_ context.Context, queryVec []float64, topK int) ([]SearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]SearchResult, 0, len(m.docs))
	for _, doc := range m.docs {
		if len(doc.Embedding) == 0 {
			continue
		}
		score := cosineSimilarity(queryVec, doc.Embedding)
		results = append(results, SearchResult{Document: doc, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

func (m *MemoryStore) Delete(_ context.Context, ids []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	filtered := m.docs[:0]
	for _, doc := range m.docs {
		if !idSet[doc.ID] {
			filtered = append(filtered, doc)
		}
	}
	m.docs = filtered
	return nil
}

func (m *MemoryStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.docs)
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}
