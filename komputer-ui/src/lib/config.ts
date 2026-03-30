// Runtime config loaded from /config.js (window.__KOMPUTER_CONFIG__)
// This file is served as a static asset and can be replaced via ConfigMap mount.

interface KomputerConfig {
  apiUrl: string;
}

declare global {
  interface Window {
    __KOMPUTER_CONFIG__?: KomputerConfig;
  }
}

const defaultConfig: KomputerConfig = {
  apiUrl: "http://localhost:8080",
};

export function getConfig(): KomputerConfig {
  if (typeof window !== "undefined" && window.__KOMPUTER_CONFIG__) {
    return window.__KOMPUTER_CONFIG__;
  }
  return defaultConfig;
}
