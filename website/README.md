# goemail documentation site

The source for <https://kartikrocks.github.io/goemail/>, built with React 19 +
TypeScript + Vite + Tailwind v4 + Shiki.

It lives on `main` alongside the code on purpose: a PR that changes documented
behaviour should change the docs in the same review. Docs on a separate branch
drift from the code they describe.

## Local development

```bash
npm ci
npm run dev       # http://localhost:5173/goemail/
```

The dev server is configured with `base: '/goemail/'` so URLs match the
deployed Pages path; the trailing slash matters.

## Before pushing

```bash
npm run lint      # eslint
npm run build     # tsc -b && vite build
```

## Editing docs

Each topic is a React component under `src/content/` (not Markdown). Add a
new topic by creating a `src/content/<topic>.tsx` file and wiring it into the
sidebar in `src/components/Sidebar.tsx`.

## Deployment

Pushing to `main` with changes under `website/` deploys via
`.github/workflows/docs.yml`. There is nothing to run by hand — do not commit
built output to a branch.
