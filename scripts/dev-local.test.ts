import { expect, test } from 'bun:test';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';

// Stub external commands so reset coverage can never touch a real database.
function runReset(databaseURL: string) {
  const dir = mkdtempSync(join(tmpdir(), 'nuchi-reset-test-'));
  const calls = join(dir, 'calls');
  try {
    for (const command of ['docker', 'go']) {
      writeFileSync(
        join(dir, command),
        '#!/bin/sh\nprintf "%s\\n" "$*" >> "$RESET_TEST_CALLS"\n',
        { mode: 0o755 }
      );
    }
    const result = Bun.spawnSync(
      [process.execPath, resolve('scripts/dev-local.ts'), '--reset-db'],
      {
        env: {
          ...process.env,
          PATH: dir,
          RESET_TEST_CALLS: calls,
          DATABASE_URL: databaseURL,
          POSTGRES_PORT: '54329',
        },
        stdout: 'pipe',
        stderr: 'pipe',
      }
    );
    let commands = '';
    try {
      commands = readFileSync(calls, 'utf8');
    } catch {
      // Validation should reject invalid targets before spawning any command.
    }
    return { ...result, commands };
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
}

test('database reset recreates and migrates before exiting without starting the API', () => {
  const result = runReset(
    'postgres://nuchi:nuchi@localhost:54329/nuchi?sslmode=disable'
  );
  expect(result.exitCode).toBe(0);
  const commands = result.commands;
  expect(commands).toContain('dropdb -U postgres --if-exists --force nuchi');
  expect(commands.indexOf('dropdb')).toBeLessThan(commands.indexOf('createdb'));
  expect(commands.indexOf('createdb')).toBeLessThan(
    commands.indexOf('ALTER SCHEMA')
  );
  expect(commands.indexOf('ALTER SCHEMA')).toBeLessThan(
    commands.indexOf('goose')
  );
  expect(commands).toContain('-dir migrations postgres');
  expect(commands).not.toContain('build -o');
});

for (const target of [
  'postgres://nuchi:nuchi@remote.example:54329/nuchi',
  'postgres://nuchi:nuchi@localhost:5432/nuchi',
  'postgres://nuchi:nuchi@localhost:54329/other',
  'postgres://postgres:postgres@localhost:54329/nuchi',
  'postgres://nuchi:nuchi@localhost:54329/nuchi?host=remote.example',
]) {
  test(`database reset rejects unsafe target ${target}`, () => {
    const result = runReset(target);
    expect(result.exitCode).not.toBe(0);
    expect(result.commands).toBe('');
  });
}
