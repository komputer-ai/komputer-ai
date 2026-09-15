import { describe, it, expect } from 'vitest';
import { isBedrockModelId, isVertexModelId } from './model-utils';

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

  it('returns false for pinned Vertex ids', () => {
    // Vertex ids and Bedrock ids are mutually exclusive.
    expect(isBedrockModelId('claude-sonnet-4-5@20250929')).toBe(false);
  });
});

describe('isVertexModelId', () => {
  it('returns false for empty string', () => {
    expect(isVertexModelId('')).toBe(false);
  });

  it('returns false for friendly Anthropic-API names', () => {
    expect(isVertexModelId('claude-sonnet-4-6')).toBe(false);
    expect(isVertexModelId('claude-opus-4-7')).toBe(false);
  });

  it('returns true for pinned Vertex model ids', () => {
    expect(isVertexModelId('claude-sonnet-4-5@20250929')).toBe(true);
    expect(isVertexModelId('claude-haiku-4-5@20251001')).toBe(true);
    expect(isVertexModelId('claude-opus-4-1@20250805')).toBe(true);
  });

  it('returns false for Bedrock inference profiles and ARNs', () => {
    expect(isVertexModelId('us.anthropic.claude-sonnet-4-5-20250929-v1:0')).toBe(false);
    expect(
      isVertexModelId('arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.anthropic.claude-sonnet-4-6'),
    ).toBe(false);
  });

  it('returns false for a bare "@" with no version date', () => {
    expect(isVertexModelId('claude-sonnet-4-5@')).toBe(false);
  });
});
