"use client";

import { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/kit/dialog";
import { Button } from "@/components/kit/button";
import { Input } from "@/components/kit/input";
import { getApiToken, setApiToken, UNAUTHORIZED_EVENT } from "@/lib/auth";

// Asks for the API token when the API answers 401 (its auth is on).
export function ApiTokenDialog() {
  const [open, setOpen] = useState(false);
  const [token, setToken] = useState("");

  useEffect(() => {
    const onUnauthorized = () => {
      setToken(getApiToken());
      setOpen(true);
    };
    window.addEventListener(UNAUTHORIZED_EVENT, onUnauthorized);
    return () => window.removeEventListener(UNAUTHORIZED_EVENT, onUnauthorized);
  }, []);

  const save = () => {
    setApiToken(token.trim());
    setOpen(false);
    window.location.reload();
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="max-w-md mx-auto bg-[var(--color-surface)] border-[var(--color-border)] text-[var(--color-text)]">
        <DialogHeader>
          <DialogTitle className="text-[var(--color-text)]">API token required</DialogTitle>
          <DialogDescription className="text-[var(--color-text-secondary)]">
            This komputer API requires a token. It is saved in this browser only.
          </DialogDescription>
        </DialogHeader>
        <form
          className="mt-4"
          onSubmit={(e) => {
            e.preventDefault();
            save();
          }}
        >
          <Input
            type="password"
            autoComplete="off"
            placeholder="API token"
            value={token}
            onChange={(e) => setToken(e.target.value)}
            autoFocus
          />
          <DialogFooter className="mt-4">
            <Button
              type="button"
              variant="secondary"
              className="border-[var(--color-border)] text-[var(--color-text-secondary)]"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button type="submit" variant="primary" disabled={!token.trim()}>
              Save
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
