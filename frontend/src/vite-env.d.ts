/// <reference types="vite/client" />

declare global {
  interface Window {
    notify?: (msg: string, type?: 'info' | 'success' | 'warning' | 'error') => void;
  }
}

export {};