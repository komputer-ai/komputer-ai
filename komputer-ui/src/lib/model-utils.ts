/**
 * Helpers for classifying model identifier strings.
 *
 * The UI lets users type any model id into the picker (see `custom-models.ts`).
 * Some of those strings target AWS Bedrock — friendly names that the SDK
 * routes via the Anthropic API look like `claude-sonnet-4-6`, while Bedrock
 * inference-profile ids look like `us.anthropic.claude-sonnet-4-5-20250929-v1:0`
 * or full ARNs. The pickers surface this distinction so the user can tell
 * which backend their selection actually targets.
 */

/**
 * True when `value` is a Bedrock inference-profile id or model ARN.
 *
 * - ARNs (`arn:aws:bedrock:…`) cover provisioned-throughput model handles.
 * - Inference profiles are prefixed by geographic scope: `us.`, `eu.`,
 *   `apac.`, `global.` — followed by `anthropic.…`.
 *
 * Returns `false` for empty input and for friendly names like
 * `claude-sonnet-4-6`.
 */
export function isBedrockModelId(value: string): boolean {
  if (!value) return false;
  if (value.startsWith("arn:")) return true;
  return /^(us|eu|apac|global)\.anthropic\./.test(value);
}
