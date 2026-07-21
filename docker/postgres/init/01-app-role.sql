-- Runs once, on first initialization of the postgres data volume
-- (/docker-entrypoint-initdb.d), as the bootstrap superuser.
--
-- The application role must NOT be the bootstrap superuser. Superusers
-- bypass row level security entirely — FORCE included — so an app
-- connecting as one would have the RLS ownership policies in
-- backend/migrations/00003_finance_rls.sql silently not apply, and the RLS
-- tests would pass while proving nothing. PostgreSQL also refuses to let
-- the bootstrap superuser drop its own SUPERUSER attribute, so the
-- separation has to exist from the start rather than be undone later.
--
-- This mirrors the CI service in .github/workflows/ci.yml.

CREATE ROLE nuchi LOGIN PASSWORD 'nuchi';

-- The app role owns the schema, so every table the goose migrations create
-- is owned by it — which is what makes FORCE ROW LEVEL SECURITY apply.
ALTER SCHEMA public OWNER TO nuchi;

-- Installed here as superuser so the migrations do not depend on citext
-- being a trusted extension. 00001_auth_base.sql uses
-- CREATE EXTENSION IF NOT EXISTS, so it is a no-op there.
CREATE EXTENSION IF NOT EXISTS citext;
