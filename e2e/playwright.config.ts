import { defineConfig } from '@playwright/test';
import { igAndroidProfile, igIosProfile } from './fixtures/ig-profiles';

// Minimal Node env typing — @types/node is intentionally not an e2e dep.
declare const process: { env: Record<string, string | undefined> };

// workers: 1 is MANDATORY — all e2e traffic egresses from one container IP and
// RateLimitClaims is a per-IP token bucket (10 burst + 1/6s refill,
// backend/internal/middleware/rate_limit.go:70). Parallel workers would 429.
// retries: 1 covers infra flake for non-claim specs; claim specs override to
// retries: 0 in their own file (fresh waffle is seeded per spec in beforeEach).
export default defineConfig({
  testDir: './specs',
  workers: 1,
  retries: 1,
  use: {
    baseURL: process.env.BASE_URL ?? 'http://app:8383',
  },
  reporter: [
    ['list'],
    ['junit', { outputFile: 'test-results/junit.xml' }],
  ],
  projects: [
    {
      name: 'ig-ios',
      use: {
        ...igIosProfile.deviceMetrics,
        userAgent: igIosProfile.userAgent,
      },
    },
    {
      name: 'ig-android',
      use: {
        ...igAndroidProfile.deviceMetrics,
        userAgent: igAndroidProfile.userAgent,
      },
    },
  ],
});
