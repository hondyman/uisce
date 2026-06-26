# React + Vite

This template provides a minimal setup to get React working in Vite with HMR and some ESLint rules.

Currently, two official plugins are available:

- [@vitejs/plugin-react](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react) uses [Babel](https://babeljs.io/) for Fast Refresh
- [@vitejs/plugin-react-swc](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react-swc) uses [SWC](https://swc.rs/) for Fast Refresh

## Expanding the ESLint configuration

If you are developing a production application, we recommend using TypeScript with type-aware lint rules enabled. Check out the [TS template](https://github.com/vitejs/vite/tree/main/packages/create-vite/template-react-ts) for information on how to integrate TypeScript and [`typescript-eslint`](https://typescript-eslint.io) in your project.

## Local setup notes

If you pulled recent changes that add `zustand` and other dependencies, run:

```bash
cd frontend
npm install   # or pnpm install
npm run dev
```

If `npm run dev` exits with code 130, check your shell for signals or review the terminal output for the error details.

## Environment variables (frontend)

The frontend uses Vite, so environment variables for the browser should use the `VITE_*` prefix. Examples:

```bash
VITE_API_BASE_URL=http://localhost:29080
VITE_GRAPHQL_ENDPOINT=http://localhost:8080/v1/graphql
```

To make code compatible with legacy setups that may still reference `REACT_APP_*`, use the `getEnv()` helper in TypeScript which reads either legacy keys or `VITE_*` keys (prefers Vite). The goal is to fully migrate to `VITE_*` across frontend code and docs.

```ts
import { getEnv } from '@/utils/getEnv';

const apiUrl = getEnv('', 'VITE_API_BASE_URL', 'http://localhost:29080');
```
