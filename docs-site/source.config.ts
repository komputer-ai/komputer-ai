import { defineDocs, defineConfig } from 'fumadocs-mdx/config';
import { remarkDocLinks } from './src/lib/remark-doc-links.mjs';

export const docs = defineDocs({
  dir: 'content/docs',
});

export default defineConfig({
  mdxOptions: {
    remarkPlugins: [remarkDocLinks],
  },
});
