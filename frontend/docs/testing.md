# Testing in SemLayer

This project uses multiple test runners by design. This keeps large legacy Jest suites stable while allowing fast, scoped Vitest tests for new components.

## Runners & purposes

- **Jest** — primary test runner for most existing frontend tests (legacy and large suites). Run with `npm test` from repo root (invokes `test:jest`).
- **Vitest** — fast, focused tests for new components. Only runs tests placed in `src/vitest/**` (see config). Run with `npm run test:vitest`.
- **Playwright** — end-to-end tests. Run with `npm run test:e2e`.

## Commands

From the repo root:

- `npm test` → runs Jest across workspaces (default)
- `npm run test:jest` → explicit Jest run
- `npm run test:vitest` → runs Vitest (only `src/vitest/**/*` by config)
- `npm run test:e2e` → runs Playwright tests

Note: In the `frontend/` package we also have `npm run test:vitest` to run the frontend vitest lane locally.

## Adding new tests

Guidelines:

- If the test uses `jest.mock`, `jest.fn`, or depends on existing Jest setup (i18n mocks, complex shared setup) → add it under `__tests__` or `src/**/__tests__` and it will be run by Jest.
- If the test is a small, fast, DOM-focused component test and can use `vi.mock` / `vi.fn` → put it under `src/vitest/` and it will be run by Vitest.
- If the test is an end-to-end flow, put it under `e2e/` and use Playwright.

## Vitest configuration

Vitest is configured in `frontend/vitest.config.ts` and only includes `src/vitest/**/*.test.{ts,tsx}`. This prevents Vitest from running legacy Jest tests.

## Troubleshooting

- If you see `ReferenceError: jest is not defined` — you are running Vitest on Jest tests. Move the test to Jest or port mocks to `vi.*` and move to `src/vitest/`.
- If Playwright tests fail under Vitest — run them with `npm run test:e2e`.

If you'd like, I can create a short migration checklist for moving safe tests from Jest → Vitest (3–5 at a time).