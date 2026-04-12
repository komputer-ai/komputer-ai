"""Integration tests for skills."""

import pytest
from komputer_ai.models import CreateSkillRequest, PatchSkillRequest


SKILL_NAME = "sdk-test-skill"


@pytest.fixture(autouse=True)
def cleanup(client):
    _safe_delete(client, SKILL_NAME)
    yield
    _safe_delete(client, SKILL_NAME)


def _safe_delete(client, name):
    try:
        client.skills.delete_skill(name)
    except Exception:
        pass


class TestSkills:
    def test_create_skill(self, client):
        req = CreateSkillRequest(
            name=SKILL_NAME,
            description="Test skill from SDK",
            content="echo 'hello from sdk test'",
        )
        resp = client.skills.create_skill(req)
        assert resp.name == SKILL_NAME
        assert resp.description == "Test skill from SDK"

    def test_list_skills_contains_created(self, client):
        req = CreateSkillRequest(
            name=SKILL_NAME,
            description="list test",
            content="echo list",
        )
        client.skills.create_skill(req)

        skills = client.skills.list_skills()
        names = [s.name for s in skills]
        assert SKILL_NAME in names

    def test_get_skill(self, client):
        req = CreateSkillRequest(
            name=SKILL_NAME,
            description="get test",
            content="echo get",
        )
        client.skills.create_skill(req)

        resp = client.skills.get_skill(SKILL_NAME)
        assert resp.name == SKILL_NAME
        assert resp.content == "echo get"

    def test_patch_skill(self, client):
        req = CreateSkillRequest(
            name=SKILL_NAME,
            description="original",
            content="echo original",
        )
        client.skills.create_skill(req)

        patch = PatchSkillRequest(description="updated description")
        resp = client.skills.patch_skill(SKILL_NAME, patch)
        assert resp.description == "updated description"

    def test_delete_skill(self, client):
        req = CreateSkillRequest(
            name=SKILL_NAME, description="delete me", content="echo bye"
        )
        client.skills.create_skill(req)

        client.skills.delete_skill(SKILL_NAME)

        with pytest.raises(Exception):
            client.skills.get_skill(SKILL_NAME)
