-- infra/migrations/001_create_doc_vectors.sql
-- Ensure pgvector extension is installed; using ankane/pgvector image is easiest for Docker
CREATE EXTENSION IF NOT EXISTS vector;
CREATE TABLE IF NOT EXISTS doc_vectors (
  id TEXT PRIMARY KEY,
  content TEXT,
  embedding vector(1536),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_doc_vectors_embedding ON doc_vectors USING ivfflat (embedding vector_l2_ops) WITH (lists = 100);