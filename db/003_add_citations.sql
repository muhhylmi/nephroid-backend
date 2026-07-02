-- Add citations JSONB column to permanently save RAG chunks in chat history
ALTER TABLE messages ADD COLUMN IF NOT EXISTS citations JSONB;
