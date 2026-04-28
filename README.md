# goemail website

This branch contains the source for the goemail documentation site, deployed to GitHub Pages.

The library source lives on the [`main`](https://github.com/KARTIKrocks/goemail/tree/main) branch.

## Local development

```bash
cd goemail-website
npm install
npm run dev
```

The dev server runs at `http://localhost:5173/goemail/`. Vite is configured with `base: '/goemail/'`
so URLs match the deployed Pages path; the trailing slash matters.

## Build

```bash
cd goemail-website
npm run build
npm run preview   # serves the built dist/ folder locally
```

## Deployment

Pushing to the `website` branch triggers `.github/workflows/deploy.yml`, which builds the Vite
project and publishes `goemail-website/dist` to GitHub Pages.

To enable Pages for the repository:

1. Go to **Settings → Pages**
2. Under **Build and deployment**, set **Source** to **GitHub Actions**

After the first successful workflow run, the site will be available at
`https://kartikrocks.github.io/goemail/`.
