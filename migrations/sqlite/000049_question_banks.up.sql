CREATE TABLE IF NOT EXISTS question_banks (
    id                         TEXT PRIMARY KEY,
    tenant_id                  INTEGER NOT NULL,
    name                       TEXT NOT NULL,
    description                TEXT NOT NULL DEFAULT '',
    source_knowledge_base_ids  TEXT NOT NULL DEFAULT '[]',
    single_choice_count        INTEGER NOT NULL DEFAULT 5,
    multiple_choice_count      INTEGER NOT NULL DEFAULT 3,
    short_answer_count         INTEGER NOT NULL DEFAULT 2,
    difficulty                 TEXT NOT NULL DEFAULT '',
    attempt_question_count     INTEGER NOT NULL DEFAULT 10,
    grading_prompt             TEXT NOT NULL DEFAULT '',
    status                     TEXT NOT NULL,
    generation_error           TEXT NOT NULL DEFAULT '',
    generation_model_id        TEXT NOT NULL DEFAULT '',
    creator_id                 TEXT NOT NULL DEFAULT '',
    created_at                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_question_banks_tenant_status_updated
    ON question_banks (tenant_id, status, updated_at DESC);

CREATE TABLE IF NOT EXISTS question_bank_questions (
    id                          TEXT PRIMARY KEY,
    tenant_id                   INTEGER NOT NULL,
    bank_id                     TEXT NOT NULL REFERENCES question_banks(id) ON DELETE CASCADE,
    type                        TEXT NOT NULL,
    prompt                      TEXT NOT NULL,
    options                     TEXT NOT NULL DEFAULT '[]',
    correct_options             TEXT NOT NULL DEFAULT '[]',
    reference_answer            TEXT NOT NULL DEFAULT '',
    explanation                 TEXT NOT NULL DEFAULT '',
    source_knowledge_base_id    TEXT NOT NULL DEFAULT '',
    source_knowledge_base_name  TEXT NOT NULL DEFAULT '',
    source_knowledge_id         TEXT NOT NULL DEFAULT '',
    source_chunk_id             TEXT NOT NULL DEFAULT '',
    source_title                TEXT NOT NULL DEFAULT '',
    sort_order                  INTEGER NOT NULL DEFAULT 0,
    created_at                  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_question_bank_questions_bank_order
    ON question_bank_questions (tenant_id, bank_id, sort_order);

CREATE TABLE IF NOT EXISTS question_bank_attempts (
    id                       TEXT PRIMARY KEY,
    tenant_id                INTEGER NOT NULL,
    bank_id                  TEXT NOT NULL REFERENCES question_banks(id) ON DELETE CASCADE,
    user_id                  TEXT NOT NULL,
    status                   TEXT NOT NULL,
    question_count           INTEGER NOT NULL DEFAULT 0,
    score                    REAL NOT NULL DEFAULT 0,
    correct_count            INTEGER NOT NULL DEFAULT 0,
    items                    TEXT NOT NULL DEFAULT '[]',
    grading_prompt_snapshot  TEXT NOT NULL DEFAULT '',
    model_id                 TEXT NOT NULL DEFAULT '',
    started_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    submitted_at             DATETIME NULL,
    created_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_question_bank_attempts_user_history
    ON question_bank_attempts (tenant_id, bank_id, user_id, started_at DESC);
