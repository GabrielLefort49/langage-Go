-- Full text index works without pgvector.
CREATE TABLE IF NOT EXISTS notes (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
  enrichment_status TEXT NOT NULL DEFAULT 'pending',
  short_summary TEXT,
  score DOUBLE PRECISION
);

CREATE TABLE IF NOT EXISTS note_tags (
  note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  tag TEXT NOT NULL,
  PRIMARY KEY (note_id, tag)
);

CREATE TABLE IF NOT EXISTS note_embeddings (
  note_id TEXT PRIMARY KEY REFERENCES notes(id) ON DELETE CASCADE,
  embedding TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS notes_fulltext_idx ON notes USING GIN (to_tsvector('english', coalesce(title,'') || ' ' || coalesce(content,'')));
CREATE INDEX IF NOT EXISTS notes_status_idx ON notes (enrichment_status);
