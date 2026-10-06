import { describe, it, expect } from 'vitest';
import {
  mcpEndpointUrl,
  isLocalUrl,
  claudeCodeAddCommand,
  claudeDesktopConfig,
} from './mcp-connect';

describe('mcpEndpointUrl', () => {
  it('strips a trailing slash from an absolute apiUrl and appends /mcp', () => {
    expect(mcpEndpointUrl('http://localhost:8080/', 'https://ignored.example.com')).toBe(
      'http://localhost:8080/mcp',
    );
  });

  it('appends /mcp to an absolute apiUrl with no trailing slash', () => {
    expect(mcpEndpointUrl('https://api.komputer.example.com', 'https://ignored.example.com')).toBe(
      'https://api.komputer.example.com/mcp',
    );
  });

  it('resolves an empty apiUrl against origin', () => {
    expect(mcpEndpointUrl('', 'https://komputer.example.com')).toBe(
      'https://komputer.example.com/mcp',
    );
  });

  it('resolves a relative apiUrl against origin', () => {
    expect(mcpEndpointUrl('/api', 'https://komputer.example.com')).toBe(
      'https://komputer.example.com/api/mcp',
    );
  });

  it('strips a trailing slash on a relative apiUrl resolved against origin', () => {
    expect(mcpEndpointUrl('/api-proxy/', 'https://komputer.example.com')).toBe(
      'https://komputer.example.com/api-proxy/mcp',
    );
  });
});

describe('isLocalUrl', () => {
  it('is true for localhost', () => {
    expect(isLocalUrl('http://localhost:8080/mcp')).toBe(true);
  });

  it('is true for 127.0.0.1', () => {
    expect(isLocalUrl('http://127.0.0.1:8080/mcp')).toBe(true);
  });

  it('is true for ::1', () => {
    expect(isLocalUrl('http://[::1]:8080/mcp')).toBe(true);
  });

  it('is true for 0.0.0.0', () => {
    expect(isLocalUrl('http://0.0.0.0:8080/mcp')).toBe(true);
  });

  it('is false for a public host', () => {
    expect(isLocalUrl('https://komputer.example.com/mcp')).toBe(false);
  });

  it('does not throw and returns false for garbage input', () => {
    expect(isLocalUrl('not a url at all')).toBe(false);
  });
});

describe('claudeCodeAddCommand', () => {
  const url = 'http://localhost:8080/mcp';

  it('has no --scope flag for "local"', () => {
    expect(claudeCodeAddCommand(url, 'local')).toBe(
      'claude mcp add --transport http komputer http://localhost:8080/mcp',
    );
  });

  it('inserts --scope user right after add', () => {
    expect(claudeCodeAddCommand(url, 'user')).toBe(
      'claude mcp add --scope user --transport http komputer http://localhost:8080/mcp',
    );
  });

  it('inserts --scope project right after add', () => {
    expect(claudeCodeAddCommand(url, 'project')).toBe(
      'claude mcp add --scope project --transport http komputer http://localhost:8080/mcp',
    );
  });
});

describe('claudeDesktopConfig', () => {
  it('produces valid JSON with no --allow-http for an https url', () => {
    const json = claudeDesktopConfig('https://komputer.example.com/mcp');
    expect(() => JSON.parse(json)).not.toThrow();
    expect(JSON.parse(json)).toEqual({
      mcpServers: {
        komputer: {
          command: 'npx',
          args: ['-y', 'mcp-remote', 'https://komputer.example.com/mcp'],
        },
      },
    });
  });

  it('appends --allow-http for an http url', () => {
    const json = claudeDesktopConfig('http://localhost:8080/mcp');
    expect(JSON.parse(json)).toEqual({
      mcpServers: {
        komputer: {
          command: 'npx',
          args: ['-y', 'mcp-remote', 'http://localhost:8080/mcp', '--allow-http'],
        },
      },
    });
  });

  it('pretty-prints with 2-space indentation', () => {
    const json = claudeDesktopConfig('https://komputer.example.com/mcp');
    expect(json).toContain('\n  "mcpServers"');
  });
});
