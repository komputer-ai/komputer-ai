import { describe, it, expect } from 'vitest';
import { isBedrockModelId } from './model-utils';

describe('isBedrockModelId', () => {
  it('returns false for empty string', () => {
    expect(isBedrockModelId('')).toBe(false);
  });

  it('returns false for friendly Anthropic-API names', () => {
    expect(isBedrockModelId('claude-sonnet-4-6')).toBe(false);
    expect(isBedrockModelId('claude-opus-4-7')).toBe(false);
    expect(isBedrockModelId('claude-haiku-4-5-20251001')).toBe(false);
  });

  it('returns true for us-region inference profiles', () => {
    expect(isBedrockModelId('us.anthropic.claude-sonnet-4-5-20250929-v1:0')).toBe(true);
    expect(isBedrockModelId('us.anthropic.claude-sonnet-4-6')).toBe(true);
  });

  it('returns true for eu-region inference profiles', () => {
    expect(isBedrockModelId('eu.anthropic.claude-sonnet-4-5-20250929-v1:0')).toBe(true);
  });

  it('returns true for apac inference profiles', () => {
    expect(isBedrockModelId('apac.anthropic.claude-sonnet-4-5-20250929-v1:0')).toBe(true);
  });

  it('returns true for global inference profiles', () => {
    expect(isBedrockModelId('global.anthropic.claude-opus-4-7')).toBe(true);
  });

  it('returns true for ARNs', () => {
    expect(
      isBedrockModelId(
        'arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.anthropic.claude-sonnet-4-5-20250929-v1:0',
      ),
    ).toBe(true);
  });

  it('returns false for arbitrary "us." strings that are not anthropic models', () => {
    // The "anthropic." segment is load-bearing — guards against false positives.
    expect(isBedrockModelId('us.something-else')).toBe(false);
    expect(isBedrockModelId('us.anthropos.fake')).toBe(false);
  });
});
