"""Integration tests for skills."""

import pytest


SKILL_NAME = "sdk-test-skill"


@pytest.fixture(autouse=True)
def cleanup(client):
    _safe_delete(client, SKILL_NAME)
    yield
    _safe_delete(client, SKILL_NAME)


def _safe_delete(client, name):
    try:
        client.delete_skill(name)
    except Exception:
        pass


class TestSkills:
    def test_create_skill(self, client):
        resp = client.create_skill(
            name=SKILL_NAME,
            description="Test skill from SDK",
            content="echo 'hello from sdk test'",
        )
        assert resp.name == SKILL_NAME
        assert resp.description == "Test skill from SDK"

    def test_list_skills_contains_created(self, client):
        client.create_skill(
            name=SKILL_NAME,
            description="list test",
            content="echo list",
        )

        skills = client.list_skills()
        names = [s.name for s in skills]
        assert SKILL_NAME in names

    def test_get_skill(self, client):
        client.create_skill(
            name=SKILL_NAME,
            description="get test",
            content="echo get",
        )

        resp = client.get_skill(SKILL_NAME)
        assert resp.name == SKILL_NAME
        assert resp.content == "echo get"

    def test_patch_skill(self, client):
        client.create_skill(
            name=SKILL_NAME,
            description="original",
            content="echo original",
        )

        resp = client.patch_skill(SKILL_NAME, description="updated description")
        assert resp.description == "updated description"

    def test_delete_skill(self, client):
        client.create_skill(
            name=SKILL_NAME,
            description="delete me",
            content="echo bye",
        )

        client.delete_skill(SKILL_NAME)

        with pytest.raises(Exception):
            client.get_skill(SKILL_NAME)
