"""Tests for SDK model serialization and deserialization."""

import pytest

from komputer_ai.models import (
    AgentListResponse,
    AgentResponse,
    ConnectorResponse,
    CreateAgentRequest,
    CreateConnectorRequest,
    CreateMemoryRequest,
    CreateScheduleAgentSpec,
    CreateScheduleRequest,
    CreateSecretRequest,
    CreateSkillRequest,
    MemoryResponse,
    OfficeListResponse,
    OfficeMemberResponse,
    OfficeResponse,
    PatchAgentRequest,
    PatchMemoryRequest,
    PatchScheduleRequest,
    PatchSkillRequest,
    ScheduleListResponse,
    ScheduleResponse,
    SecretListResponse,
    SecretResponse,
    SkillResponse,
    UpdateSecretRequest,
)


# --- Agent models ---


class TestCreateAgentRequest:
    def test_required_fields(self):
        req = CreateAgentRequest(name="my-agent", instructions="do stuff")
        assert req.name == "my-agent"
        assert req.instructions == "do stuff"

    def test_all_fields(self):
        req = CreateAgentRequest(
            name="my-agent",
            instructions="do stuff",
            model="claude-sonnet-4-6",
            template_ref="custom-template",
            role="manager",
            namespace="prod",
            secret_refs=["secret-1", "secret-2"],
            memories=["mem-1"],
            skills=["skill-1"],
            connectors=["conn-1"],
            lifecycle="Sleep",
            office_manager="boss-agent",
            system_prompt="You are a helpful assistant.",
        )
        d = req.to_dict()
        assert d["name"] == "my-agent"
        assert d["model"] == "claude-sonnet-4-6"
        assert d["secretRefs"] == ["secret-1", "secret-2"]
        assert d["lifecycle"] == "Sleep"
        assert d["systemPrompt"] == "You are a helpful assistant."

    def test_json_round_trip(self):
        req = CreateAgentRequest(
            name="test-agent",
            instructions="run tests",
            model="claude-sonnet-4-6",
            skills=["skill-a"],
        )
        json_str = req.to_json()
        restored = CreateAgentRequest.from_json(json_str)
        assert restored.name == req.name
        assert restored.instructions == req.instructions
        assert restored.model == req.model
        assert restored.skills == req.skills


class TestAgentResponse:
    def test_basic_fields(self):
        resp = AgentResponse(
            name="agent-1",
            namespace="default",
            model="claude-sonnet-4-6",
            status="Running",
            created_at="2025-01-01T00:00:00Z",
        )
        assert resp.name == "agent-1"
        assert resp.status == "Running"

    def test_optional_fields(self):
        resp = AgentResponse(
            name="agent-1",
            namespace="default",
            model="claude-sonnet-4-6",
            status="Idle",
            task_status="completed",
            last_task_message="Done",
            last_task_cost_usd="0.05",
            total_cost_usd="1.23",
            total_tokens=50000,
            secrets=["API_KEY"],
            memories=["mem-1"],
            skills=["skill-1"],
            connectors=["slack"],
            instructions="do stuff",
            system_prompt="custom prompt",
            created_at="2025-01-01T00:00:00Z",
        )
        d = resp.to_dict()
        assert d["taskStatus"] == "completed"
        assert d["totalTokens"] == 50000
        assert d["systemPrompt"] == "custom prompt"
        assert d["secrets"] == ["API_KEY"]

    def test_json_round_trip(self):
        resp = AgentResponse(
            name="agent-1",
            namespace="default",
            model="claude-sonnet-4-6",
            status="Running",
            total_tokens=1000,
            created_at="2025-01-01T00:00:00Z",
        )
        restored = AgentResponse.from_json(resp.to_json())
        assert restored.name == resp.name
        assert restored.total_tokens == resp.total_tokens


class TestAgentListResponse:
    def test_with_agents(self):
        agent = AgentResponse(
            name="a1", namespace="default", model="m", status="Running", created_at="t"
        )
        lst = AgentListResponse(agents=[agent])
        assert len(lst.agents) == 1
        assert lst.agents[0].name == "a1"

    def test_empty_list(self):
        lst = AgentListResponse(agents=[])
        assert lst.agents == []


class TestPatchAgentRequest:
    def test_partial_update(self):
        req = PatchAgentRequest(model="claude-sonnet-4-6", lifecycle="AutoDelete")
        d = req.to_dict()
        assert d["model"] == "claude-sonnet-4-6"
        assert d["lifecycle"] == "AutoDelete"

    def test_empty_patch(self):
        req = PatchAgentRequest()
        d = req.to_dict()
        # All fields should be absent (omitempty)
        assert "model" not in d or d.get("model") is None


# --- Memory models ---


class TestCreateMemoryRequest:
    def test_required_fields(self):
        req = CreateMemoryRequest(name="context", content="important info")
        assert req.name == "context"
        assert req.content == "important info"

    def test_json_round_trip(self):
        req = CreateMemoryRequest(
            name="ctx", content="data", description="desc", namespace="prod"
        )
        restored = CreateMemoryRequest.from_json(req.to_json())
        assert restored.name == req.name
        assert restored.description == req.description


class TestMemoryResponse:
    def test_fields(self):
        resp = MemoryResponse(
            name="mem-1",
            namespace="default",
            content="stored content",
            description="a memory",
            attached_agents=2,
            agent_names=["a1", "a2"],
            created_at="2025-01-01T00:00:00Z",
        )
        assert resp.attached_agents == 2
        assert resp.agent_names == ["a1", "a2"]


# --- Skill models ---


class TestCreateSkillRequest:
    def test_required_fields(self):
        req = CreateSkillRequest(
            name="deploy", description="deploy to prod", content="#!/bin/bash\necho deploy"
        )
        assert req.name == "deploy"
        assert req.description == "deploy to prod"

    def test_json_round_trip(self):
        req = CreateSkillRequest(
            name="s", description="d", content="c", namespace="ns"
        )
        restored = CreateSkillRequest.from_json(req.to_json())
        assert restored.content == "c"


class TestSkillResponse:
    def test_fields(self):
        resp = SkillResponse(
            name="skill-1",
            namespace="default",
            description="desc",
            content="script",
            attached_agents=1,
            agent_names=["a1"],
            is_default=True,
            created_at="2025-01-01T00:00:00Z",
        )
        assert resp.is_default is True


# --- Schedule models ---


class TestCreateScheduleRequest:
    def test_required_fields(self):
        req = CreateScheduleRequest(
            name="daily-check",
            schedule="0 9 * * *",
            instructions="check health",
        )
        assert req.name == "daily-check"
        assert req.schedule == "0 9 * * *"

    def test_with_agent_spec(self):
        agent_spec = CreateScheduleAgentSpec(
            model="claude-sonnet-4-6", lifecycle="AutoDelete"
        )
        req = CreateScheduleRequest(
            name="sched",
            schedule="*/5 * * * *",
            instructions="ping",
            agent=agent_spec,
            timezone="UTC",
            auto_delete=True,
        )
        assert req.agent.model == "claude-sonnet-4-6"
        assert req.auto_delete is True


class TestScheduleResponse:
    def test_fields(self):
        resp = ScheduleResponse(
            name="sched-1",
            namespace="default",
            schedule="0 9 * * *",
            phase="Active",
            run_count=10,
            successful_runs=9,
            failed_runs=1,
            total_cost_usd="5.00",
            created_at="2025-01-01T00:00:00Z",
        )
        assert resp.run_count == 10
        assert resp.successful_runs == 9


# --- Secret models ---


class TestCreateSecretRequest:
    def test_required_fields(self):
        req = CreateSecretRequest(
            name="api-keys", data={"API_KEY": "sk-123", "TOKEN": "tok-456"}
        )
        assert req.name == "api-keys"
        assert req.data["API_KEY"] == "sk-123"


class TestSecretResponse:
    def test_fields(self):
        resp = SecretResponse(
            name="secret-1",
            namespace="default",
            keys=["API_KEY", "TOKEN"],
            managed=True,
            attached_agents=1,
            agent_names=["a1"],
            created_at="2025-01-01T00:00:00Z",
        )
        assert resp.keys == ["API_KEY", "TOKEN"]
        assert resp.managed is True


# --- Connector models ---


class TestCreateConnectorRequest:
    def test_required_fields(self):
        req = CreateConnectorRequest(
            name="slack-conn",
            service="slack",
            url="https://mcp.slack.com",
        )
        assert req.service == "slack"

    def test_with_auth(self):
        req = CreateConnectorRequest(
            name="gh-conn",
            service="github",
            url="https://mcp.github.com",
            auth_type="token",
            auth_secret_name="gh-token",
            auth_secret_key="TOKEN",
        )
        d = req.to_dict()
        assert d["authType"] == "token"
        assert d["authSecretName"] == "gh-token"


class TestConnectorResponse:
    def test_fields(self):
        resp = ConnectorResponse(
            name="conn-1",
            namespace="default",
            service="slack",
            url="https://mcp.slack.com",
            type="remote",
            attached_agents=2,
            agent_names=["a1", "a2"],
            created_at="2025-01-01T00:00:00Z",
        )
        assert resp.attached_agents == 2


# --- Office models ---


class TestOfficeResponse:
    def test_fields(self):
        member = OfficeMemberResponse(
            name="worker-1", role="worker", task_status="completed"
        )
        resp = OfficeResponse(
            name="office-1",
            namespace="default",
            manager="manager-1",
            phase="Active",
            total_agents=3,
            active_agents=2,
            completed_agents=1,
            total_cost_usd="10.00",
            members=[member],
            created_at="2025-01-01T00:00:00Z",
        )
        assert resp.total_agents == 3
        assert len(resp.members) == 1
        assert resp.members[0].name == "worker-1"


class TestOfficeListResponse:
    def test_with_offices(self):
        office = OfficeResponse(
            name="o1", namespace="default", created_at="t"
        )
        lst = OfficeListResponse(offices=[office])
        assert len(lst.offices) == 1
