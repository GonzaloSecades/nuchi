import { describe, expect, it } from 'bun:test';

// Deliberate type error, pushed to prove the #83 typecheck gate fails a job.
// The test itself passes at runtime. Reverted after CI goes red.
describe('gate proof', () => {
  it('passes at runtime but does not typecheck', () => {
    const n: number = 'not a number';
    expect(n).toBeDefined();
  });
});
