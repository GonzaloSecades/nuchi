import { defineConfig, globalIgnores } from 'eslint/config';
import nextVitals from 'eslint-config-next/core-web-vitals';
import nextTs from 'eslint-config-next/typescript';
import eslintConfigPrettier from 'eslint-config-prettier';

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,
  // Override default ignores of eslint-config-next.
  globalIgnores([
    // Default ignores of eslint-config-next:
    '.next/**',
    'out/**',
    'build/**',
    'next-env.d.ts',
    // Sibling git worktrees. The documented workflow checks them out inside the
    // repo, so their build output and vendored code are visible from the root
    // config — `.next/**` above only covers the root's own build. Without this,
    // linting master reports tens of thousands of problems that belong to
    // another branch and takes minutes.
    '.worktrees/**',
    '.claude/worktrees/**',
  ]),
  // Disable ESLint rules that conflict with Prettier
  eslintConfigPrettier,
]);

export default eslintConfig;
