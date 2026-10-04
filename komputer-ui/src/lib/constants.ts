export const MODELS = [
  // Current models.
  { value: "claude-fable-5-1", label: "claude-fable-5-1" },
  { value: "claude-opus-5-5", label: "claude-opus-5-5" },
  { value: "claude-sonnet-5-5", label: "claude-sonnet-5-5" },
  { value: "claude-haiku-4-5", label: "claude-haiku-4-5" },
  // Legacy models, still available — newest first.
  { value: "claude-fable-5", label: "claude-fable-5" },
  { value: "claude-opus-5", label: "claude-opus-5" },
  { value: "claude-opus-4-8", label: "claude-opus-4-8" },
  { value: "claude-opus-4-7", label: "claude-opus-4-7" },
  { value: "claude-opus-4-6", label: "claude-opus-4-6" },
  { value: "claude-opus-4-5", label: "claude-opus-4-5" },
  { value: "claude-sonnet-5", label: "claude-sonnet-5" },
  { value: "claude-sonnet-4-6", label: "claude-sonnet-4-6" },
];

export const LIFECYCLES = [
  { value: "default", label: "Default — keep running" },
  { value: "Sleep", label: "Sleep — preserve workspace" },
  { value: "AutoDelete", label: "Auto Delete — one-shot" },
];
