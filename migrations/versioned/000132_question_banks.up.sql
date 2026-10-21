CREATE TABLE IF NOT EXISTS question_banks (
    id                         VARCHAR(36) PRIMARY KEY,
    tenant_id                  BIGINT NOT NULL,
    name                       VARCHAR(255) NOT NULL,
    description                TEXT NOT NULL DEFAULT '',
    source_knowledge_base_ids  JSONB NOT NULL DEFAULT '[]'::jsonb,
    single_choice_count        INTEGER NOT NULL DEFAULT 5,
    multiple_choice_count      INTEGER NOT NULL DEFAULT 3,
    short_answer_count         INTEGER NOT NULL DEFAULT 2,
    difficulty                 VARCHAR(24) NOT NULL DEFAULT '',
    attempt_question_count     INTEGER NOT NULL DEFAULT 10,
    grading_prompt             TEXT NOT NULL DEFAULT '',
    status                     VARCHAR(24) NOT NULL,
    generation_error           TEXT NOT NULL DEFAULT '',
    generation_model_id        VARCHAR(64) NOT NULL DEFAULT '',
    creator_id                 VARCHAR(512) NOT NULL DEFAULT '',
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_question_banks_tenant_status_updated
    ON question_banks (tenant_id, status, updated_at DESC);

CREATE TABLE IF NOT EXISTS question_bank_questions (
    id                          VARCHAR(36) PRIMARY KEY,
    tenant_id                   BIGINT NOT NULL,
    bank_id                     VARCHAR(36) NOT NULL REFERENCES question_banks(id) ON DELETE CASCADE,
    type                        VARCHAR(24) NOT NULL,
    prompt                      TEXT NOT NULL,
    options                     JSONB NOT NULL DEFAULT '[]'::jsonb,
    correct_options             JSONB NOT NULL DEFAULT '[]'::jsonb,
    reference_answer            TEXT NOT NULL DEFAULT '',
    explanation                 TEXT NOT NULL DEFAULT '',
    source_knowledge_base_id    VARCHAR(36) NOT NULL DEFAULT '',
    source_knowledge_base_name  VARCHAR(255) NOT NULL DEFAULT '',
    source_knowledge_id         VARCHAR(36) NOT NULL DEFAULT '',
    source_chunk_id             VARCHAR(36) NOT NULL DEFAULT '',
    source_title                VARCHAR(512) NOT NULL DEFAULT '',
    sort_order                  INTEGER NOT NULL DEFAULT 0,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_question_bank_questions_bank_order
    ON question_bank_questions (tenant_id, bank_id, sort_order);

CREATE TABLE IF NOT EXISTS question_bank_attempts (
    id                       VARCHAR(36) PRIMARY KEY,
    tenant_id                BIGINT NOT NULL,
    bank_id                  VARCHAR(36) NOT NULL REFERENCES question_banks(id) ON DELETE CASCADE,
    user_id                  VARCHAR(512) NOT NULL,
    status                   VARCHAR(24) NOT NULL,
    question_count           INTEGER NOT NULL DEFAULT 0,
    score                    DOUBLE PRECISION NOT NULL DEFAULT 0,
    correct_count            INTEGER NOT NULL DEFAULT 0,
    items                    JSONB NOT NULL DEFAULT '[]'::jsonb,
    grading_prompt_snapshot  TEXT NOT NULL DEFAULT '',
    model_id                 VARCHAR(64) NOT NULL DEFAULT '',
    started_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitted_at             TIMESTAMPTZ NULL,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_question_bank_attempts_user_history
    ON question_bank_attempts (tenant_id, bank_id, user_id, started_at DESC);
