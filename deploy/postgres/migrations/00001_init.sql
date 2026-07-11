-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- LLM Bindings
-- ============================================================
CREATE TABLE IF NOT EXISTS llm_bindings (
    id                  UUID            PRIMARY KEY,
    tenant_id           VARCHAR(255)    NOT NULL,
    name                VARCHAR(255)    NOT NULL,
    description         TEXT,
    provider            VARCHAR(50)     NOT NULL,
    api_base_url        VARCHAR(500)    NOT NULL,
    api_key_secret_ref  VARCHAR(255)    NOT NULL,
    default_model       VARCHAR(255)    NOT NULL,
    default_temperature DOUBLE PRECISION NOT NULL,
    default_max_tokens  INTEGER         NOT NULL,
    timeout_seconds     INTEGER         NOT NULL,
    status              VARCHAR(50)     NOT NULL,
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_llm_bindings_tenant_id ON llm_bindings(tenant_id);
-- Partial unique index: allow name reuse after soft-delete (status='inactive')
CREATE UNIQUE INDEX IF NOT EXISTS unique_llm_bindings_tenant_name_active
    ON llm_bindings (tenant_id, name) WHERE status <> 'inactive';

-- ============================================================
-- MCP Clients
-- ============================================================
CREATE TABLE IF NOT EXISTS mcp_clients (
    id              UUID            PRIMARY KEY,
    tenant_id       VARCHAR(255)    NOT NULL,
    name            VARCHAR(255)    NOT NULL,
    description     TEXT,
    transport       VARCHAR(50)     NOT NULL,
    command         VARCHAR(500),
    args            JSONB,
    env             JSONB,
    url             VARCHAR(500),
    auth_secret_ref VARCHAR(255),
    status          VARCHAR(50)     NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_mcp_clients_tenant_name UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_mcp_clients_tenant_id ON mcp_clients(tenant_id);

-- ============================================================
-- Skills & Skill Collections
-- ============================================================
CREATE TABLE IF NOT EXISTS skills (
    id          UUID            PRIMARY KEY,
    tenant_id   VARCHAR(255)    NOT NULL,
    name        VARCHAR(255)    NOT NULL,
    description TEXT,
    status      VARCHAR(50)     NOT NULL,
    content     TEXT,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_skills_tenant_name UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_skills_tenant_id ON skills(tenant_id);

CREATE TABLE IF NOT EXISTS skill_collections (
    id          UUID            PRIMARY KEY,
    tenant_id   VARCHAR(255)    NOT NULL,
    name        VARCHAR(255)    NOT NULL,
    description TEXT,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_skill_collections_tenant_name UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_skill_collections_tenant_id ON skill_collections(tenant_id);

CREATE TABLE IF NOT EXISTS skill_collection_skills (
    skill_collection_id UUID REFERENCES skill_collections(id) ON DELETE CASCADE,
    skill_id            UUID REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (skill_collection_id, skill_id)
);

-- ============================================================
-- Prompt Templates, Collections & Versions
-- ============================================================
CREATE TABLE IF NOT EXISTS prompt_templates (
    id          UUID            PRIMARY KEY,
    tenant_id   VARCHAR(255)    NOT NULL,
    name        VARCHAR(255)    NOT NULL,
    content     TEXT            NOT NULL,
    status      VARCHAR(50)     NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_prompt_templates_tenant_name UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_prompt_templates_tenant_id ON prompt_templates(tenant_id);

CREATE TABLE IF NOT EXISTS prompt_versions (
    id               UUID    PRIMARY KEY,
    prompt_id        UUID    NOT NULL REFERENCES prompt_templates(id) ON DELETE CASCADE,
    version_number   INT     NOT NULL,
    content_snapshot TEXT    NOT NULL,
    created_at       TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_prompt_id_version UNIQUE (prompt_id, version_number)
);
CREATE INDEX IF NOT EXISTS idx_prompt_versions_prompt_id ON prompt_versions(prompt_id);

CREATE TABLE IF NOT EXISTS prompt_collections (
    id          UUID            PRIMARY KEY,
    tenant_id   VARCHAR(255)    NOT NULL,
    name        VARCHAR(255)    NOT NULL,
    description TEXT,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_prompt_collections_tenant_name UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_prompt_collections_tenant_id ON prompt_collections(tenant_id);

CREATE TABLE IF NOT EXISTS prompt_collection_templates (
    prompt_collection_id UUID REFERENCES prompt_collections(id) ON DELETE CASCADE,
    prompt_template_id   UUID REFERENCES prompt_templates(id)   ON DELETE CASCADE,
    PRIMARY KEY (prompt_collection_id, prompt_template_id)
);

-- ============================================================
-- Communities
-- ============================================================
CREATE TABLE IF NOT EXISTS communities (
    id            UUID                PRIMARY KEY,
    tenant_id     VARCHAR(255)        NOT NULL,
    name          VARCHAR(255)        NOT NULL,
    description   TEXT,
    topology      VARCHAR(50)         NOT NULL DEFAULT 'single-agent',
    configuration JSONB               NOT NULL DEFAULT '{}',
    status        VARCHAR(50)         NOT NULL DEFAULT 'created',
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at    TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT uq_tenant_community_name UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_communities_tenant_id ON communities(tenant_id);

-- ============================================================
-- Agents
-- (no prompt_template column — replaced by agent_prompts association table)
-- ============================================================
CREATE TABLE IF NOT EXISTS agents (
    id                UUID            PRIMARY KEY,
    tenant_id         VARCHAR(255)    NOT NULL,
    name              VARCHAR(255)    NOT NULL,
    description       TEXT,
    role              VARCHAR(50)     NOT NULL DEFAULT 'spoke',
    brain             JSONB           NOT NULL,
    short_term_memory JSONB           NOT NULL,
    long_term_memory  JSONB           NOT NULL,
    mcp_clients       JSONB           NOT NULL,
    status            VARCHAR(50)     NOT NULL,
    tier              VARCHAR(50)     NOT NULL DEFAULT '',
    created_at        TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at        TIMESTAMP WITH TIME ZONE NOT NULL,
    community_id      UUID REFERENCES communities(id) ON DELETE RESTRICT,
    CONSTRAINT unique_agents_tenant_name UNIQUE (tenant_id, name),
    CONSTRAINT check_agent_brain CHECK (
        (brain->>'llm_binding_id') IS NOT NULL AND (brain->>'llm_binding_id') <> ''
    )
);
CREATE INDEX IF NOT EXISTS idx_agents_tenant_id ON agents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_agents_community ON agents(community_id);

-- ============================================================
-- Agent — Skill associations
-- ============================================================
CREATE TABLE IF NOT EXISTS agent_skills (
    agent_id UUID REFERENCES agents(id) ON DELETE CASCADE,
    skill_id UUID REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (agent_id, skill_id)
);

-- ============================================================
-- Agent — Prompt associations (ordered, multi-prompt support)
-- ============================================================
CREATE TABLE IF NOT EXISTS agent_prompts (
    agent_id           UUID NOT NULL REFERENCES agents(id)           ON DELETE CASCADE,
    prompt_template_id UUID NOT NULL REFERENCES prompt_templates(id) ON DELETE CASCADE,
    position           INT  NOT NULL,
    PRIMARY KEY (agent_id, prompt_template_id)
);
CREATE INDEX IF NOT EXISTS idx_agent_prompts_agent_id ON agent_prompts(agent_id);

-- ============================================================
-- Agent — Prompt Collection associations (ordered)
-- ============================================================
CREATE TABLE IF NOT EXISTS agent_prompt_collections (
    agent_id             UUID NOT NULL REFERENCES agents(id)              ON DELETE CASCADE,
    prompt_collection_id UUID NOT NULL REFERENCES prompt_collections(id)  ON DELETE CASCADE,
    position             INT  NOT NULL,
    PRIMARY KEY (agent_id, prompt_collection_id)
);
CREATE INDEX IF NOT EXISTS idx_agent_prompt_collections_agent_id ON agent_prompt_collections(agent_id);

-- ============================================================
-- Agent Registrations
-- ============================================================
CREATE TABLE IF NOT EXISTS agent_registrations (
    agent_id     UUID            NOT NULL REFERENCES agents(id)      ON DELETE CASCADE,
    community_id UUID            NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    tenant_id    VARCHAR(255)    NOT NULL,
    card         JSONB           NOT NULL,
    last_seen_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (agent_id, community_id)
);
CREATE INDEX IF NOT EXISTS idx_agent_registrations_tenant_id ON agent_registrations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_agent_registrations_community ON agent_registrations(community_id);

-- ============================================================
-- Community Assignments
-- ============================================================
CREATE TABLE IF NOT EXISTS community_assignments (
    community_id UUID            NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    agent_id     UUID            NOT NULL REFERENCES agents(id)      ON DELETE CASCADE,
    tenant_id    VARCHAR(255)    NOT NULL,
    role         TEXT            NOT NULL,
    informed_at  TIMESTAMPTZ,
    assigned_at  TIMESTAMPTZ     NOT NULL,
    PRIMARY KEY (community_id, agent_id)
);
CREATE INDEX IF NOT EXISTS idx_community_assignments_agent_id  ON community_assignments(agent_id);
CREATE INDEX IF NOT EXISTS idx_community_assignments_tenant_id ON community_assignments(tenant_id);

-- Deprecated backward-compat: keep agents.role populated via trigger
CREATE OR REPLACE FUNCTION sync_agent_role_from_assignment()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    UPDATE agents SET role = NEW.role WHERE id = NEW.agent_id;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_sync_agent_role
AFTER INSERT OR UPDATE ON community_assignments
FOR EACH ROW EXECUTE FUNCTION sync_agent_role_from_assignment();

-- ============================================================
-- Seed: Hub system prompt template (system-locked UUID)
-- ============================================================
INSERT INTO prompt_templates (id, tenant_id, name, content, status, created_at)
VALUES (
    'ffffffff-0000-0000-0000-000000000001',
    'system',
    'hub-system-prompt',
    '{{if .Description}}{{.Description}}{{else}}You are a helpful orchestrator agent.{{end}}

You have access to the following specialized Spoke agents in this community:
{{.Spokes}}

To coordinate the conversation, you must output a valid JSON response specifying your next step.
- To delegate tasks to Spoke subagents concurrently, output:
  {"action": "delegate", "spokes": [{"spoke": "<agent_name>", "message": "<task description>"}, ...] }
- If a specialized Spoke agent asks a clarifying question to the user or indicates that information/details are missing, you must immediately choose the "finalize" action and return that question directly to the user so they can reply.
- If you have completed the user request and want to finalize the response, output:
  {"action": "finalize", "response": "<final response message to the user>"}
- Do not delegate the wait state or try to delegate again if you are waiting for user input.

Dynamic Routing & Delegation Guidelines:
1. Carefully inspect the Name and Description of the available Spoke agents.
2. During the information-gathering phase of a thread (where details are missing or clarifying questions are needed), delegate tasks ONLY to agents whose descriptions indicate they perform inquiry, question-asking, coaching, or detail gathering.
3. Do NOT delegate tasks to synthesis, compiling, or final-answer agents (e.g., agents whose descriptions state they summarize findings or produce final outputs) during the information-gathering phase. Only delegate to them once all details are fully gathered and you are ready to produce the final findings.

Response Synthesis Guidelines:
1. Messages prefixed with "[Observation]" in the conversation history are responses received from Spoke agents — they are NOT user messages.
2. When finalizing, you MUST synthesize and integrate the Spoke observations into a single cohesive, polished response for the user.
3. Do NOT copy-paste or concatenate Spoke responses verbatim. Rewrite and merge them into a well-structured answer.',
    'active',
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- Seed version 1 for the hub system prompt
INSERT INTO prompt_versions (id, prompt_id, version_number, content_snapshot, created_at)
SELECT
    gen_random_uuid(),
    id,
    1,
    content,
    created_at
FROM prompt_templates
WHERE id = 'ffffffff-0000-0000-0000-000000000001'
ON CONFLICT DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER  IF EXISTS trg_sync_agent_role ON community_assignments;
DROP FUNCTION IF EXISTS sync_agent_role_from_assignment();
DROP TABLE IF EXISTS community_assignments;
DROP TABLE IF EXISTS agent_registrations;
DROP TABLE IF EXISTS agent_prompt_collections;
DROP TABLE IF EXISTS agent_prompts;
DROP TABLE IF EXISTS agent_skills;
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS communities;
DROP TABLE IF EXISTS prompt_collection_templates;
DROP TABLE IF EXISTS prompt_versions;
DROP TABLE IF EXISTS prompt_collections;
DROP TABLE IF EXISTS prompt_templates;
DROP TABLE IF EXISTS skill_collection_skills;
DROP TABLE IF EXISTS skill_collections;
DROP TABLE IF EXISTS skills;
DROP TABLE IF EXISTS mcp_clients;
DROP TABLE IF EXISTS llm_bindings;
-- +goose StatementEnd
