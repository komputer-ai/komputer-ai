import type { ReactNode } from 'react';
import type { Metadata } from 'next';
import { RootProvider } from 'fumadocs-ui/provider/next';
import './global.css';

const BASE = 'https://komputer-ai.github.io/komputer-ai';

export const metadata: Metadata = {
  metadataBase: new URL(`${BASE}/`),
  title: {
    default: 'Komputer.AI — Distributed Claude AI agents on Kubernetes',
    template: '%s — Komputer.AI',
  },
  description:
    'Open-source, Kubernetes-native platform for running persistent Claude AI agents. Built on CRDs, operators, and the Kubernetes API. Stream agent output via REST + WebSocket.',
  applicationName: 'Komputer.AI',
  keywords: [
    'Claude',
    'Anthropic',
    'AI agents',
    'Kubernetes',
    'agent platform',
    'MCP',
    'autonomous agents',
    'CRD',
    'Kubernetes operator',
    'open source',
  ],
  authors: [{ name: 'Komputer.AI', url: 'https://github.com/komputer-ai/komputer-ai' }],
  creator: 'Komputer.AI',
  publisher: 'Komputer.AI',
  alternates: { canonical: '/' },
  openGraph: {
    type: 'website',
    siteName: 'Komputer.AI',
    title: 'Komputer.AI — Distributed Claude AI agents on Kubernetes',
    description:
      'Open-source, Kubernetes-native platform for running persistent Claude AI agents. Built on CRDs and operators.',
    url: '/',
    images: [
      {
        url: 'og-image.jpg',
        width: 1200,
        height: 630,
        type: 'image/jpeg',
        alt: 'Komputer.AI — Distributed Claude AI agents on Kubernetes',
      },
    ],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Komputer.AI — Distributed Claude AI agents on Kubernetes',
    description: 'Open-source platform for running persistent Claude AI agents on Kubernetes.',
    images: ['og-image.jpg'],
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      'max-snippet': -1,
      'max-image-preview': 'large',
      'max-video-preview': -1,
    },
  },
  icons: { icon: '/komputer-ai/icon.png' },
};

const structuredData = {
  '@context': 'https://schema.org',
  '@graph': [
    {
      '@type': 'Organization',
      '@id': `${BASE}/#org`,
      name: 'Komputer.AI',
      url: BASE,
      logo: `${BASE}/logo-no-bg.png`,
      sameAs: ['https://github.com/komputer-ai/komputer-ai'],
    },
    {
      '@type': 'WebSite',
      '@id': `${BASE}/#site`,
      url: BASE,
      name: 'Komputer.AI',
      description: 'Distributed Claude AI agents on Kubernetes.',
      publisher: { '@id': `${BASE}/#org` },
      inLanguage: 'en',
    },
    {
      '@type': 'SoftwareApplication',
      name: 'Komputer.AI',
      applicationCategory: 'DeveloperApplication',
      operatingSystem: 'Kubernetes',
      description:
        'Open-source, Kubernetes-native platform for running persistent Claude AI agents.',
      url: BASE,
      offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
    },
  ],
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(structuredData) }}
        />
      </head>
      <body className="flex flex-col min-h-screen">
        <RootProvider>{children}</RootProvider>
      </body>
    </html>
  );
}
