"use client";

import { useState, useEffect } from "react";

/**
 * Returns true only after `delay` ms have passed while `loading` is true.
 * Prevents skeleton placeholders from flashing on fast loads, while anything
 * slower shows them almost at once instead of a blank page.
 */
export function useDelayedLoading(loading: boolean, delay = 300): boolean {
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
