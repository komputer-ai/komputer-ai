// Puts an open dropdown menu above or below its trigger. Call it from a layout
// effect: it writes straight onto the element, so the menu lands in place before
// the first paint (no show-below-then-jump, no extra render). It opens upward only
// when the menu doesn't fit below and there's more room above.
export function placeMenu(menu: HTMLElement, trigger: HTMLElement, gap = 4) {
  const rect = trigger.getBoundingClientRect();
  const spaceBelow = window.innerHeight - rect.bottom;
  const up = spaceBelow < menu.offsetHeight + gap * 2 && rect.top > spaceBelow;
  menu.style.top = up ? "auto" : "100%";
  menu.style.bottom = up ? "100%" : "auto";
  menu.style.marginTop = up ? "0" : `${gap}px`;
  menu.style.marginBottom = up ? `${gap}px` : "0";
}
