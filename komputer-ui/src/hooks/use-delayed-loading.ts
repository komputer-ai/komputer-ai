"use client";

import { useState, useEffect } from "react";

/**
 * Returns true only after `delay` ms have passed while `loading` is true.
 * Prevents skeleton placeholders from flashing on fast loads.
 */
export function useDelayedLoading(loading: boolean, delay = 2000): boolean {
  const [showLoading, setShowLoading] = useState(false);

  useEffect(() => {
    if (!loading) {
      setShowLoading(false);
      return;
    }

    const timer = setTimeout(() => setShowLoading(true), delay);
    return () => clearTimeout(timer);
  }, [loading, delay]);

  return showLoading;
}
