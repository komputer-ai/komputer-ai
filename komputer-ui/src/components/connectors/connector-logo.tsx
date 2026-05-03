"use client";

import { useEffect, useState } from "react";

/**
 * Renders a SimpleIcons-style connector logo, swapping `/white` ↔ `/black` to
 * stay legible in both dark and light themes.
 *
 * Backend stores `https://cdn.simpleicons.org/<service>/white` (tuned for
 * dark surfaces). At render time we inspect the `data-theme` attribute on
 * <html> and rewrite the URL accordingly. We also subscribe to attribute
 * changes via MutationObserver so the icon reacts when the user toggles
 * the theme without a reload.
 */
export function ConnectorLogo({
  src,
  alt,
  className,
}: {
  src: string;
  alt: string;
  className?: string;
}) {
  const [isLight, setIsLight] = useState(false);

  useEffect(() => {
    if (typeof document === "undefined") return;
    const read = () => setIsLight(document.documentElement.getAttribute("data-theme") === "light");
    read();
    const obs = new MutationObserver(read);
    obs.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });
    return () => obs.disconnect();
  }, []);

  const finalSrc = isLight ? src.replace(/\/white$/, "/black") : src;
  return <img src={finalSrc} alt={alt} className={className} />;
}
