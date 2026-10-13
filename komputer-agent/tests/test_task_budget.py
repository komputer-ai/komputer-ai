"""Tests for the per-task hard budget resolution (agent._resolve_task_budget_usd).

A per-request value (from server.py's /task handler, forwarded by the API) must
win over the KOMPUTER_TASK_BUDGET_USD env var the operator sets at pod startup —
that's what lets a budget raised or lowered after pod startup apply to the very
next task without waiting for a restart.
"""
from agent import _resolve_task_budget_usd


def test_per_request_value_wins_over_env(monkeypatch):
    monkeypatch.setenv("KOMPUTER_TASK_BUDGET_USD", "1.00")
    assert _resolve_task_budget_usd(2.50) == 2.50


def test_falls_back_to_env_var_when_no_request_value(monkeypatch):
    monkeypatch.setenv("KOMPUTER_TASK_BUDGET_USD", "1.00")
    assert _resolve_task_budget_usd(None) == 1.00


def test_no_cap_when_neither_is_set(monkeypatch):
    monkeypatch.delenv("KOMPUTER_TASK_BUDGET_USD", raising=False)
    assert _resolve_task_budget_usd(None) is None


def test_empty_env_var_means_no_cap(monkeypatch):
    monkeypatch.setenv("KOMPUTER_TASK_BUDGET_USD", "")
    assert _resolve_task_budget_usd(None) is None


def test_unparseable_env_var_means_no_cap(monkeypatch):
    monkeypatch.setenv("KOMPUTER_TASK_BUDGET_USD", "not-a-number")
    assert _resolve_task_budget_usd(None) is None


def test_zero_request_value_is_honored_not_treated_as_unset(monkeypatch):
    # 0.0 is falsy in Python but a meaningful "no budget" is represented by None,
    # not 0 — the API/operator never send 0 (parseBudget rejects it), but the
    # resolution itself must distinguish "explicitly 0" from "not provided"
    # using `is not None`, not a truthiness check.
    monkeypatch.setenv("KOMPUTER_TASK_BUDGET_USD", "5.00")
    assert _resolve_task_budget_usd(0.0) == 0.0
