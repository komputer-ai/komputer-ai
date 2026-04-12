"""Integration tests for offices."""

import pytest


class TestOffices:
    def test_list_offices(self, client):
        offices = client.list_offices()
        assert offices is not None
        assert isinstance(offices.offices, list)

    def test_get_office_not_found(self, client):
        with pytest.raises(Exception):
            client.get_office("nonexistent-office")
