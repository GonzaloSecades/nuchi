import { randomBytes } from 'node:crypto';
import { spawn } from 'node:child_process';
import { existsSync, unlinkSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

/**
 * Starts the complete local dev stack:
 *
 * - Docker Compose Postgres + Mailpit
 * - goose migrations
 * - Go API
 * - Next dev server
 *
 * The frontend still never connects to Postgres directly. This script only
 * sequences the backend-owned setup so a fresh checkout can start from one
 * command: `bun dev`.
 */
const COMPOSE_SERVICES = ['postgres', 'mailpit'] as const;
const HEALTH_RETRIES = 30;
const HEALTH_DELAY_MS = 1000;
const API_HEALTH_RETRIES = 30;
const API_HEALTH_DELAY_MS = 1000;
// Keep this aligned with .github/workflows/ci.yml's installed goose version.
const GOOSE_VERSION = 'v3.27.2';
const POSTGRES_PORT = process.env.POSTGRES_PORT ?? '54329';
const DATABASE_URL =
  process.env.DATABASE_URL ??
  `postgres://nuchi:nuchi@localhost:${POSTGRES_PORT}/nuchi?sslmode=disable`;
const GO_API_URL =
  process.env.GO_API_URL ??
  `http://localhost:${process.env.BACKEND_PORT ?? '8080'}`;
const GENERATED_AUTH_SECRET = process.env.AUTH_JWT_SECRET === undefined;
const AUTH_JWT_SECRET =
  process.env.AUTH_JWT_SECRET ?? randomBytes(48).toString('base64');
const API_BIN = join(
  tmpdir(),
  `nuchi-api-dev-${process.pid}${process.platform === 'win32' ? '.exe' : ''}`
);

const baseEnv = {
  ...process.env,
  APP_ENV: process.env.APP_ENV ?? 'local',
  APP_BASE_URL: process.env.APP_BASE_URL ?? 'http://localhost:3000',
  AUTH_COOKIE_SECURE: process.env.AUTH_COOKIE_SECURE ?? 'false',
  AUTH_JWT_SECRET,
  COMPOSE_PROJECT_NAME: process.env.COMPOSE_PROJECT_NAME ?? 'nuchi',
  DATABASE_URL,
  GO_API_URL,
  MAIL_FROM: process.env.MAIL_FROM ?? 'nuchi@localhost',
  POSTGRES_PORT,
  SMTP_ADDR: process.env.SMTP_ADDR ?? 'localhost:1025',
};

const children = new Set<ReturnType<typeof spawn>>();
let stopping = false;

async function sleep(ms: number) {
  await new Promise((resolve) => setTimeout(resolve, ms));
}

function log(message: string) {
  console.log(`[dev] ${message}`);
}

function spawnLogged(
  command: [string, ...string[]],
  options: { cwd?: string } = {}
) {
  const proc = spawn(command[0], command.slice(1), {
    cwd: options.cwd,
    env: baseEnv,
    stdio: 'inherit',
  });

  children.add(proc);
  proc.once('exit', () => {
    children.delete(proc);
  });

  return proc;
}

async function run(
  command: [string, ...string[]],
  options: { cwd?: string; quiet?: boolean } = {}
) {
  const code = await new Promise<number | null>((resolve, reject) => {
    const proc = spawn(command[0], command.slice(1), {
      cwd: options.cwd,
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
 * Refuses to run the local stack against anything but the Compose Postgres.
 *
 * This script runs `goose up` unattended, so a wrong `DATABASE_URL` is not a
 * failed connection — it is migrations applied to a database that never asked
 * for them. Two ways that happens, both from a stale `.env.local`:
 *
 * - A **remote** host left over from the pre-#85 Neon stack. Fatal, no
 *   override: `bun dev` provisions local Docker Postgres, so a remote target is
 *   always a mistake here. Migrate a remote database deliberately, by hand.
 * - A **local** host on the wrong port — typically `5432` from before the
 *   Compose port moved to 54329, which lands on whatever host Postgres happens
 *   to be listening there.
 */
function validateDatabaseTarget() {
  let parsed: URL;
  try {
    parsed = new URL(DATABASE_URL);
  } catch {
    throw new Error(
      `DATABASE_URL must be an absolute Postgres URL, got: ${DATABASE_URL}`
    );
  }

  // `URL.hostname` keeps the brackets on IPv6 literals, so the loopback address
  // has to be listed in its bracketed form to match at all.
  const localHosts = new Set(['localhost', '127.0.0.1', '[::1]']);
  if (!localHosts.has(parsed.hostname)) {
    throw new Error(
      [
        `DATABASE_URL points at the remote host ${parsed.hostname}, but bun dev only drives the local Docker Postgres.`,
        'It would have applied backend/migrations/ to that database. Refusing.',
        'Check DATABASE_URL in .env.local — a Neon URL from before #85 is the usual cause.',
        `Expected local default: postgres://nuchi:nuchi@localhost:${POSTGRES_PORT}/nuchi?sslmode=disable`,
      ].join('\n')
    );
  }

  const databasePort = parsed.port || '5432';
  if (databasePort !== POSTGRES_PORT) {
    throw new Error(
      [
        `DATABASE_URL points at ${parsed.hostname}:${databasePort}, but POSTGRES_PORT is ${POSTGRES_PORT}.`,
        'Update your local DATABASE_URL to match POSTGRES_PORT before running migrations.',
        `Expected local default: postgres://nuchi:nuchi@localhost:${POSTGRES_PORT}/nuchi?sslmode=disable`,
      ].join('\n')
    );
  }
}

function databaseTarget() {
  const parsed = new URL(DATABASE_URL);
  const databaseName =
    parsed.pathname.replace(/^\//, '') || '(default database)';

  return `${parsed.hostname}:${parsed.port || '5432'}/${databaseName}`;
}

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
        env: baseEnv,
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

async function waitForApi(api: ReturnType<typeof spawn>) {
  const healthUrl = new URL('/api/health', GO_API_URL);

  await new Promise<void>((resolve, reject) => {
    let settled = false;

    const settle = (callback: () => void) => {
      if (settled) {
        return;
      }
      settled = true;
      api.off('exit', onExit);
      callback();
    };

    const onExit = (code: number | null) => {
      settle(() =>
        reject(
          new Error(
            stopping
              ? 'Interrupted before the Go API became healthy'
              : `Go API exited with code ${code} before becoming healthy`
          )
        )
      );
    };

    const poll = async () => {
      for (let attempt = 1; attempt <= API_HEALTH_RETRIES; attempt += 1) {
        if (settled) {
          return;
        }

        try {
          const response = await fetch(healthUrl);
          if (response.ok) {
            settle(resolve);
            return;
          }
        } catch {
          // Keep polling while the Go process starts.
        }

        await sleep(API_HEALTH_DELAY_MS);
      }

      settle(() =>
        reject(new Error(`Go API did not become healthy at ${healthUrl}`))
      );
    };

    api.once('exit', onExit);
    void poll();
  });
}

function stopChildren() {
  if (stopping) {
    return;
  }
  stopping = true;

  for (const child of children) {
    child.kill();
  }
}

function watchProcess(
  proc: ReturnType<typeof spawn>,
  label: string,
  settle: (code: number) => void,
  reject: (reason: Error) => void
) {
  proc.once('error', reject);
  proc.once('exit', (code) => {
    if (!stopping && code !== 0) {
      reject(new Error(`${label} exited with code ${code}`));
      return;
    }

    settle(code ?? 0);
  });
}

async function main() {
  validateDatabaseTarget();

  const nextBin =
    process.platform === 'win32'
      ? join('.', 'node_modules', '.bin', 'next.cmd')
      : join('.', 'node_modules', '.bin', 'next');

  if (!existsSync(nextBin)) {
    throw new Error('Next binary not found. Run `bun install` first.');
  }

  if (GENERATED_AUTH_SECRET) {
    log(
      'Generated an ephemeral AUTH_JWT_SECRET; set one in .env.local to keep access tokens valid across restarts'
    );
  }

  log('Starting Postgres and Mailpit');
  await run(['docker', 'compose', 'up', '-d', ...COMPOSE_SERVICES]);
  await waitForPostgres();

  log(`Applying backend migrations to ${databaseTarget()}`);
  await run(
    [
      'go',
      'run',
      `github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION}`,
      '-dir',
      'migrations',
      'postgres',
      DATABASE_URL,
      'up',
    ],
    { cwd: 'backend' }
  );

  // Registered before the build, not after the spawns: from here on there is a
  // binary in tmpdir to unlink, and an interrupt during the health wait would
  // otherwise kill the script with the default handler and skip the cleanup in
  // `finally`.
  process.on('SIGINT', stopChildren);
  process.on('SIGTERM', stopChildren);

  log('Building Go API');
  await run(['go', 'build', '-o', API_BIN, './cmd/api'], { cwd: 'backend' });

  log('Starting Go API');
  const api = spawnLogged([API_BIN]);
  await waitForApi(api);

  log('Starting Next dev server');
  const next = spawnLogged([nextBin, 'dev']);

  return await new Promise<number>((resolve, reject) => {
    watchProcess(api, 'Go API', resolve, reject);
    watchProcess(next, 'Next dev server', resolve, reject);
    process.once('SIGINT', () => resolve(0));
    process.once('SIGTERM', () => resolve(0));
  });
}

main()
  .then((code) => {
    process.exitCode = code;
  })
  .catch((error) => {
    console.error(error instanceof Error ? error.message : error);
    stopChildren();
    process.exitCode = 1;
  })
  .finally(() => {
    stopChildren();
    try {
      unlinkSync(API_BIN);
    } catch {
      // The binary may not exist if setup failed before the build step.
    }
  });
