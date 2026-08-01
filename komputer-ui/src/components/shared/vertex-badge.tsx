/**
 * A tiny uppercase pill that marks a model identifier as targeting Google
 * Vertex AI. Used in the model pickers next to the selected value and option
 * rows so users can tell pinned Vertex ids (e.g. `claude-sonnet-4-5@20250929`)
 * apart from friendly Anthropic-API names at a glance.
 *
 * Blue counterpart to the amber `BedrockBadge` — same pill shape, different
 * provider hue.
 */
export function VertexBadge() {
  return (
    <span className="text-[8px] tracking-wider uppercase px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-400 leading-none shrink-0">
      Vertex
    </span>
  );
}
