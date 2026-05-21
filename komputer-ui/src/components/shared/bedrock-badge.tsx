/**
 * A tiny uppercase pill that marks a model identifier as targeting AWS
 * Bedrock. Used in the model pickers next to the selected value and option
 * rows so users can tell Bedrock inference profiles apart from friendly
 * Anthropic-API names at a glance.
 *
 * Style matches the existing amber pill convention in this codebase
 * (see `service-template-grid.tsx`, `create-connector-modal.tsx`).
 */
export function BedrockBadge() {
  return (
    <span className="text-[8px] tracking-wider uppercase px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-400 leading-none shrink-0">
      Bedrock
    </span>
  );
}
