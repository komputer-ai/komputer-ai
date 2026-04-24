import os
import pytest

import metrics


@pytest.fixture(autouse=True)
def env_setup(monkeypatch):
    monkeypatch.setenv("KOMPUTER_AGENT_NAME", "self")
    monkeypatch.setenv("KOMPUTER_METRICS_PER_AGENT", "true")


def test_steering_counter_increments(env_setup):
    metrics.init()
    metrics.record_steering()
    metrics.record_steering()
    from prometheus_client import REGISTRY
    val = REGISTRY.get_sample_value(
        "komputer_agent_steering_total", {"agent_name": "self"}
    )
    assert val == 2.0


def test_mcp_status_gauge_set(env_setup):
    metrics.init()
    metrics.set_mcp_status("slack", healthy=True)
    metrics.set_mcp_status("github", healthy=False)
    from prometheus_client import REGISTRY
    slack_val = REGISTRY.get_sample_value(
        "komputer_agent_mcp_connector_status",
        {"agent_name": "self", "connector": "slack", "status": "healthy"}
    )
    assert slack_val == 1.0
    github_val = REGISTRY.get_sample_value(
        "komputer_agent_mcp_connector_status",
        {"agent_name": "self", "connector": "github", "status": "healthy"}
    )
    assert github_val == 0.0


def test_subagent_wait_observed(env_setup):
    metrics.init()
    metrics.observe_subagent_wait(5.5)
    from prometheus_client import REGISTRY
    count = REGISTRY.get_sample_value(
        "komputer_agent_subagent_wait_seconds_count", {"agent_name": "self"}
    )
    assert count == 1.0


def test_per_agent_label_disabled_uses_empty_string(monkeypatch):
    monkeypatch.setenv("KOMPUTER_METRICS_PER_AGENT", "false")
    monkeypatch.setenv("KOMPUTER_AGENT_NAME", "actual-name")
    metrics.init()
    metrics.record_steering()
    from prometheus_client import REGISTRY
    val = REGISTRY.get_sample_value(
        "komputer_agent_steering_total", {"agent_name": ""}
    )
    assert val == 1.0
