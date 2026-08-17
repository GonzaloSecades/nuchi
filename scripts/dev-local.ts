import { randomBytes } from 'node:crypto';
import { spawn } from 'node:child_process';
import { existsSync } from 'node:fs';
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
const POSTGRES_PORT = process.env.POSTGRES_PORT ?? '54329';
const DATABASE_URL =
  process.env.DATABASE_URL ??
  `postgres://nuchi:nuchi@localhost:${POSTGRES_PORT}/nuchi?sslmode=disable`;
const GO_API_URL =
  process.env.GO_API_URL ??
  `http://localhost:${process.env.BACKEND_PORT ?? '8080'}`;

const baseEnv = {
  ...process.env,
  APP_ENV: process.env.APP_ENV ?? 'local',
  APP_BASE_URL: process.env.APP_BASE_URL ?? 'http://localhost:3000',
  AUTH_COOKIE_SECURE: process.env.AUTH_COOKIE_SECURE ?? 'false',
  AUTH_JWT_SECRET:
    process.env.AUTH_JWT_SECRET ?? randomBytes(48).toString('base64'),
  COMPOSE_PROJECT_NAME: process.env.COMPOSE_PROJECT_NAME ?? 'nuchi',
  DATABASE_URL,
  GO_API_URL,
  MAIL_FROM: process.env.MAIL_FROM ?? 'nuchi@localhost',
  NEXT_PUBLIC_API_URL:
    process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:3000',
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
  label: string,
  options: { cwd?: string } = {}
) {
  const proc = spawn(command[0], command.slice(1), {
    cwd: options.cwd,
    env: baseEnv,
    stdio: ['inherit', 'pipe', 'pipe'],
  });

  children.add(proc);

  proc.stdout?.on('data', (chunk) => {
    process.stdout.write(prefixLines(label, chunk));
  });
  proc.stderr?.on('data', (chunk) => {
    process.stderr.write(prefixLines(label, chunk));
  });
  proc.once('exit', () => {
    children.delete(proc);
  });

  return proc;
}

function prefixLines(label: string, chunk: Buffer) {
  return chunk
    .toString()
    .split(/\n/)
    .map((line, index, lines) => {
      if (index === lines.length - 1 && line === '') {
        return '';
      }
      return `[${label}] ${line}`;
    })
    .join('\n');
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

async function waitForApi() {
  const healthUrl = new URL('/api/health', GO_API_URL);

  for (let attempt = 1; attempt <= API_HEALTH_RETRIES; attempt += 1) {
    try {
      const response = await fetch(healthUrl);
      if (response.ok) {
        return;
      }
    } catch {
      // Keep polling while the Go process starts.
    }

    await sleep(API_HEALTH_DELAY_MS);
  }

  throw new Error(`Go API did not become healthy at ${healthUrl}`);
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
  settle: () => void,
  reject: (reason: Error) => void
) {
  proc.once('error', reject);
  proc.once('exit', (code) => {
    if (!stopping && code !== 0) {
      reject(new Error(`${label} exited with code ${code}`));
      return;
    }

    settle();
  });
}

async function main() {
  log('Starting Postgres and Mailpit');
  await run(['docker', 'compose', 'up', '-d', ...COMPOSE_SERVICES]);
  await waitForPostgres();

  log('Applying backend migrations');
  await run(
    [
      'go',
      'run',
      'github.com/pressly/goose/v3/cmd/goose@v3.27.2',
      '-dir',
      'migrations',
      'postgres',
      DATABASE_URL,
      'up',
    ],
    { cwd: 'backend' }
  );

  log('Starting Go API');
  const api = spawnLogged(['go', 'run', './cmd/api'], 'api', {
    cwd: 'backend',
  });
  await waitForApi();

  const nextBin =
    process.platform === 'win32'
      ? join('.', 'node_modules', '.bin', 'next.cmd')
      : join('.', 'node_modules', '.bin', 'next');

  if (!existsSync(nextBin)) {
    throw new Error('Next binary not found. Run `bun install` first.');
  }

  log('Starting Next dev server');
  const next = spawnLogged([nextBin, 'dev'], 'next');

  process.on('SIGINT', stopChildren);
  process.on('SIGTERM', stopChildren);

  await new Promise<void>((resolve, reject) => {
    watchProcess(api, 'Go API', resolve, reject);
    watchProcess(next, 'Next dev server', resolve, reject);
    process.once('SIGINT', resolve);
    process.once('SIGTERM', resolve);
  });
}

main()
  .catch((error) => {
    console.error(error instanceof Error ? error.message : error);
    stopChildren();
    process.exit(1);
  })
  .finally(() => {
    stopChildren();
  });
