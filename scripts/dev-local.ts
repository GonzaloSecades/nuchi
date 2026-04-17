import { spawn } from 'node:child_process';

const LOCAL_DATABASE_URL =
  process.env.LOCAL_DATABASE_URL ??
  'postgres://nuchi:nuchi_dev_password@127.0.0.1:54329/nuchi_dev';

const COMPOSE_FILE = 'docker-compose.dev.yml';
const SERVICE_NAME = 'nuchi-postgres';
const HEALTH_RETRIES = 30;
const HEALTH_DELAY_MS = 1000;

const baseEnv = {
  ...process.env,
  APP_ENV: 'local',
  DATABASE_URL: LOCAL_DATABASE_URL,
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

async function isPostgresReady() {
  const code = await new Promise<number | null>((resolve) => {
    const proc = spawn(
      'docker',
      [
        'compose',
        '-f',
        COMPOSE_FILE,
        'exec',
        '-T',
        SERVICE_NAME,
        'pg_isready',
        '-U',
        'nuchi',
        '-d',
        'nuchi_dev',
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
  await run([
    'docker',
    'compose',
    '-f',
    COMPOSE_FILE,
    'up',
    '-d',
    SERVICE_NAME,
  ]);
  await waitForPostgres();
  await run(['bun', './scripts/migrate.ts']);

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
