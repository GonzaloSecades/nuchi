// Deliberate type error, pushed to prove the #83 typecheck gate fails a job.
// Nothing imports this. Reverted after CI goes red.
export const definitelyNotANumber: number = 'a string';
