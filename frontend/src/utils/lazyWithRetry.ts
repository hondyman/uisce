import { lazy } from 'react';
import { devError } from './devLogger';

interface LazyRetryOptions {
  retries?: number;
  retryDelayMs?: number;
  onGiveUp?: (error: unknown) => void;
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

/**
 * Wraps React.lazy imports with a small retry/backoff strategy so transient chunk-loading errors
 * (e.g. Vite dev server restarts) don't strand the UI. Falls back to a soft reload on final failure.
 */
export function lazyWithRetry<T extends { default: React.ComponentType<any> }>(
  factory: () => Promise<T>,
  { retries = 2, retryDelayMs = 300, onGiveUp }: LazyRetryOptions = {}
) {
  return lazy(async () => {
    let attempt = 0;
    let lastError: unknown;
    while (attempt <= retries) {
      try {
        return await factory();
      } catch (error) {
        lastError = error;
        // Only retry for the classic dynamic import fetch error or network hiccups
        const message = error instanceof Error ? error.message : String(error);
        const isChunkError = /Failed to fetch dynamically imported module/i.test(message) || /ChunkLoadError/i.test(message);
        if (!isChunkError || attempt === retries) {
          break;
        }
        attempt += 1;
        await sleep(retryDelayMs * attempt);
      }
    }

    if (typeof window !== 'undefined') {
      devError('Lazy chunk failed to load after retries, reloading window.', lastError);
      if (!(window as any).__SEMLAYER_LAZY_RELOADING__) {
        (window as any).__SEMLAYER_LAZY_RELOADING__ = true;
        setTimeout(() => window.location.reload(), 150);
      }
    }

    if (onGiveUp) {
      onGiveUp(lastError);
    }

    throw lastError ?? new Error('Unknown lazy import failure');
  });
}
