"""Integration tests for schedules."""

import pytest


SCHEDULE_NAME = "sdk-test-schedule"


@pytest.fixture(autouse=True)
def cleanup(client):
    _safe_delete(client, SCHEDULE_NAME)
    yield
    _safe_delete(client, SCHEDULE_NAME)


def _safe_delete(client, name):
    try:
        client.delete_schedule(name)
    except Exception:
        pass


class TestSchedules:
    def test_create_schedule(self, client):
        resp = client.create_schedule(
            name=SCHEDULE_NAME,
            schedule="0 9 * * *",
            instructions="Run daily health check",
            timezone="UTC",
            auto_delete=True,
        )
        assert resp.name == SCHEDULE_NAME
        assert resp.schedule == "0 9 * * *"

    def test_list_schedules(self, client):
        client.create_schedule(
            name=SCHEDULE_NAME,
            schedule="0 9 * * *",
            instructions="daily check",
        )

        schedules = client.list_schedules()
        names = [s.name for s in schedules.schedules]
        assert SCHEDULE_NAME in names

    def test_get_schedule(self, client):
        client.create_schedule(
            name=SCHEDULE_NAME,
            schedule="0 9 * * *",
            instructions="daily check",
        )

        resp = client.get_schedule(SCHEDULE_NAME)
        assert resp.name == SCHEDULE_NAME
        assert resp.schedule == "0 9 * * *"

    def test_patch_schedule(self, client):
        client.create_schedule(
            name=SCHEDULE_NAME,
            schedule="0 9 * * *",
            instructions="daily check",
        )

        resp = client.patch_schedule(SCHEDULE_NAME, schedule="*/30 * * * *")
        assert resp.schedule == "*/30 * * * *"

    def test_delete_schedule(self, client):
        client.create_schedule(
            name=SCHEDULE_NAME,
            schedule="0 9 * * *",
            instructions="daily check",
        )

        client.delete_schedule(SCHEDULE_NAME)

        with pytest.raises(Exception):
            client.get_schedule(SCHEDULE_NAME)
