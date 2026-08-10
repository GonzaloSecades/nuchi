import { spawn } from 'node:child_process';

/**
 * Starts the local services the app needs, then the Next dev server.
 *
 * **This no longer runs migrations, and no longer provisions its own
 * database.** Both changed with #85, which removed Drizzle:
 *
 * - Migrations belong to goose and run from `backend/` against the Go API's
 *   database. Running them from here would mean the frontend owned a schema it
 *   no longer touches — the frontend does not connect to Postgres at all now.
 * - The old `docker-compose.dev.yml` was deleted with it. It ran a separate
 *   `nuchi_dev` database on port 54329 whose bootstrap superuser *was* the
 *   application role, which is precisely the layout `docker-compose.yml`
 *   documents as unsafe: superusers bypass row level security outright, FORCE
 *   included, so RLS policies would silently not apply. Only the deleted
 *   Drizzle migrate script ever pointed at it.
 *
 * So this brings up the same `postgres` and `mailpit` services the Go API and
 * the backend test suite use, and leaves schema and API startup to `backend/`.
 *
 * **A fresh Compose volume therefore has no tables.** The init script creates
 * only the `nuchi` role and `citext`, and the API calls `VerifyRLSActive`
 * before serving, so it exits rather than starting against an unmigrated
 * database. Run the goose step from the README's "Running Next and Go Together"
 * flow once before `go run ./cmd/api`:
 *
 *     cd backend
 *     goose -dir migrations postgres 'postgres://nuchi:nuchi@localhost:5432/nuchi?sslmode=disable' up
 *
 * The URL is literal on purpose. `$DATABASE_URL` is not exported by anything —
 * `.env.local` is loaded by Bun for this process, not for goose or the Go API —
 * so passing `"$DATABASE_URL"` sends goose an empty string and it falls back to
 * libpq defaults, failing against your OS username.
 *
 * That step is deliberately not invoked here: it would put the backend's schema
 * ownership, and a Go toolchain dependency, inside the frontend dev server.
 */
const COMPOSE_SERVICES = ['postgres', 'mailpit'] as const;
const HEALTH_RETRIES = 30;
const HEALTH_DELAY_MS = 1000;

const baseEnv = {
  ...process.env,
  APP_ENV: 'local',
};

async function sleep(ms: number) {
  await new Promise((resolve) => setTimeout(resolve, ms));
}

async function run(
  command: [string, ...string[]],
  options: { quiet?: boolean } = {}
) {
  const code = await new Promise<number | null>((resolve, reject) => {
    const proc = spawn(command[0], command.slice(1), {
      env: baseEnv,
      stdio: options.quiet ? 'ignore' : 'inherit',
    });

    proc.once('error', reject);
    proc.on('exit', resolve);
  });

  if (code !== 0) {
    throw new Error(`Command failed (${code}): ${command.join(' ')}`);
  }
}

/**
 * `pg_isready` as the bootstrap superuser, matching the compose healthcheck.
 * The application role `nuchi` is created by an init script, so probing as
 * `nuchi` would fail on a first-run volume before that script completes.
 */
async function isPostgresReady() {
  const code = await new Promise<number | null>((resolve) => {
    const proc = spawn(
      'docker',
      [
        'compose',
        'exec',
        '-T',
        'postgres',
        'pg_isready',
        '-U',
        'postgres',
        '-d',
        'nuchi',
      ],
      {
        stdio: 'ignore',
      }
    );

    proc.once('error', () => resolve(null));
    proc.on('exit', resolve);
  });

  return code === 0;
}

async function waitForPostgres() {
  for (let attempt = 1; attempt <= HEALTH_RETRIES; attempt += 1) {
    if (await isPostgresReady()) {
      return;
    }

    await sleep(HEALTH_DELAY_MS);
  }

  throw new Error('Local Postgres did not become healthy in time');
}

async function main() {
  await run(['docker', 'compose', 'up', '-d', ...COMPOSE_SERVICES]);
  await waitForPostgres();

  const nextBin =
    process.platform === 'win32'
      ? './node_modules/.bin/next.cmd'
      : './node_modules/.bin/next';

  const next = spawn(nextBin, ['dev'], {
    env: baseEnv,
    stdio: 'inherit',
  });

  const stop = () => {
    next.kill();
  };

  process.on('SIGINT', stop);
  process.on('SIGTERM', stop);

  const code = await new Promise<number | null>((resolve, reject) => {
    next.once('error', reject);
    next.on('exit', resolve);
  });

  process.exit(code ?? 0);
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : error);
  process.exit(1);
});
