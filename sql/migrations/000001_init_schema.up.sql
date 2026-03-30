BEGIN;
-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. USERS
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 2. LOCAL_CREDENTIALS
CREATE TABLE local_credentials (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    password_hash VARCHAR(255) NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 3. OAUTH_IDENTITIES
CREATE TABLE oauth_identities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL CHECK (provider IN ('google', 'github')),
    provider_user_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(provider, provider_user_id)
);

-- 4. SESSIONS
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(64) UNIQUE NOT NULL,
    user_agent TEXT,
    ip_address VARCHAR(45),
    last_used_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 5. TAGS
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    color_hex VARCHAR(7) CHECK (color_hex ~ '^#[0-9A-Fa-f]{6}$'),
    UNIQUE(user_id, name)
);

-- 6. DECISIONS
CREATE TABLE decisions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    context TEXT,
    confidence_before INTEGER CHECK (confidence_before >= 1 AND confidence_before <= 100),
    scheduled_review_date DATE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Active decisions index
CREATE INDEX idx_active_decisions_user ON decisions(user_id) WHERE deleted_at IS NULL;

-- 7. DECISION_TAGS
CREATE TABLE decision_tags (
    decision_id UUID NOT NULL REFERENCES decisions(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (decision_id, tag_id)
);

-- 8. OPTIONS
CREATE TABLE options (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    decision_id UUID NOT NULL REFERENCES decisions(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    pros TEXT,
    cons TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(decision_id, title),
    -- NOTE: this composite unique exists solely to anchor the FK in decision_outcomes,
    -- ensuring a chosen option must belong to the same decision. Do not remove.
    UNIQUE(decision_id, id)
);

-- 9. DECISION_OUTCOMES
CREATE TABLE decision_outcomes (
    decision_id UUID PRIMARY KEY REFERENCES decisions(id) ON DELETE CASCADE,
    chosen_option_id UUID NOT NULL,
    commitment_note TEXT,
    decided_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    FOREIGN KEY (decision_id, chosen_option_id) REFERENCES options(decision_id, id)
);

-- 10. EVALUATIONS
CREATE TABLE evaluations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    decision_id UUID UNIQUE NOT NULL REFERENCES decisions(id) ON DELETE CASCADE,
    satisfaction_after INTEGER CHECK (satisfaction_after >= 1 AND satisfaction_after <= 100),
    confidence_after INTEGER CHECK (confidence_after >= 1 AND confidence_after <= 100),
    hindsight_notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 11. AI_INSIGHTS
CREATE TABLE ai_insights (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    decision_id UUID NOT NULL REFERENCES decisions(id) ON DELETE CASCADE,
    model_version VARCHAR(100) NOT NULL,
    trigger_context VARCHAR(50) NOT NULL CHECK (trigger_context IN ('pre_decision', 'post_evaluation')),
    insight_text TEXT NOT NULL,
    primary_bias VARCHAR(50) CHECK (primary_bias IN ('confirmation', 'loss_aversion', 'sunk_cost', 'overconfidence', 'recency')),
    emotional_tone VARCHAR(50) CHECK (emotional_tone IN ('neutral', 'anxious', 'confident', 'uncertain', 'regretful')),
    risk_level VARCHAR(50) CHECK (risk_level IN ('low', 'medium', 'high')),
    actionability_score INTEGER CHECK (actionability_score >= 1 AND actionability_score <= 10),
    suggested_alternative TEXT,
    model_confidence REAL CHECK (model_confidence >= 0.0 AND model_confidence <= 1.0),
    generated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ============================================================
-- INDEXES
-- ============================================================

-- Sessions: user lookup and expiry cleanup job
CREATE INDEX idx_sessions_user    ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

-- OAuth: look up all identities for a user
CREATE INDEX idx_oauth_user ON oauth_identities(user_id);

-- Options: fetch options for a decision
CREATE INDEX idx_options_decision ON options(decision_id);

-- Decision tags: reverse lookup (all decisions for a tag)
CREATE INDEX idx_dtags_tag ON decision_tags(tag_id);

-- AI insights: fetch insights for a decision
CREATE INDEX idx_insights_decision ON ai_insights(decision_id);

-- ============================================================
-- TRIGGERS: keep updated_at current automatically
-- ============================================================

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_decisions_updated_at
    BEFORE UPDATE ON decisions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_local_credentials_updated_at
    BEFORE UPDATE ON local_credentials
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;