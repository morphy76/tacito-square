# TASK-M6.5.3.1: DB Migration — Association & Versioning Tables

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.3.1 |
| Status      | DRAFT |
| Spec        | SPEC-FR-M6.5.3 |
| Depends On  | none |

## Description

Create a new goose database migration under `deploy/postgres/migrations/` to introduce `prompt_versions`, `agent_prompts`, and `agent_prompt_collections` tables. Implement backfilling for existing agents and drop the old column.

## Work Items

1. **Migration File Creation**:
   - Create a migration file: `deploy/postgres/migrations/00004_prompt_collection_and_association.sql`.

2. **Migration Statements (`Up`)**:
   - Create the `prompt_versions` table:
     ```sql
     CREATE TABLE IF NOT EXISTS prompt_versions (
         id UUID PRIMARY KEY,
         prompt_id UUID NOT NULL REFERENCES prompt_templates(id) ON DELETE CASCADE,
         version_number INT NOT NULL,
         content_snapshot TEXT NOT NULL,
         created_at TIMESTAMP WITH TIME ZONE NOT NULL,
         CONSTRAINT unique_prompt_id_version UNIQUE (prompt_id, version_number)
     );
     ```
   - Create the `agent_prompts` association table:
     ```sql
     CREATE TABLE IF NOT EXISTS agent_prompts (
         agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
         prompt_template_id UUID NOT NULL REFERENCES prompt_templates(id) ON DELETE CASCADE,
         position INT NOT NULL,
         PRIMARY KEY (agent_id, prompt_template_id)
     );
     ```
   - Create the `agent_prompt_collections` association table:
     ```sql
     CREATE TABLE IF NOT EXISTS agent_prompt_collections (
         agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
         prompt_collection_id UUID NOT NULL REFERENCES prompt_collections(id) ON DELETE CASCADE,
         position INT NOT NULL,
         PRIMARY KEY (agent_id, prompt_collection_id)
     );
     ```
   - **Data Backfill**: Copy existing non-null `prompt_template` UUIDs from `agents` to `agent_prompts` table with `position = 0`.
   - **Schema Alteration**: Drop the `prompt_template` foreign key constraint and column from the `agents` table.

3. **Migration Statements (`Down`)**:
   - Re-add the `prompt_template` column to `agents` referencing `prompt_templates(id)`.
   - Backfill `prompt_template` on the `agents` table from the `agent_prompts` row where `position = 0` (or the first available prompt ID).
   - Drop the `agent_prompts`, `agent_prompt_collections`, and `prompt_versions` tables.

4. **Verification**:
   - Execute the migration up and down locally to verify correctness and safety.

## Acceptance Criteria

1. Migration `00004_prompt_collection_and_association.sql` successfully compiles and runs up/down using the goose migration runner.
2. After running Up, database tables `prompt_versions`, `agent_prompts`, and `agent_prompt_collections` exist with the defined constraints.
3. Existing agents with a configured prompt template are correctly backfilled into `agent_prompts` with `position = 0`.
4. The `prompt_template` column is successfully removed from the `agents` table.
