"""Tests for the KomputerClient wrapper and kwargs convenience methods."""

from komputer_ai.client import KomputerClient
from komputer_ai.api.agents_api import AgentsApi
from komputer_ai.api.offices_api import OfficesApi
from komputer_ai.api.schedules_api import SchedulesApi
from komputer_ai.api.memories_api import MemoriesApi
from komputer_ai.api.skills_api import SkillsApi
from komputer_ai.api.secrets_api import SecretsApi
from komputer_ai.api.connectors_api import ConnectorsApi


class TestKomputerClient:
    def test_instantiation(self):
        client = KomputerClient("http://localhost:8080")
        assert isinstance(client.agents, AgentsApi)
        assert isinstance(client.offices, OfficesApi)
        assert isinstance(client.schedules, SchedulesApi)
        assert isinstance(client.memories, MemoriesApi)
        assert isinstance(client.skills, SkillsApi)
        assert isinstance(client.secrets, SecretsApi)
        assert isinstance(client.connectors, ConnectorsApi)
        client.close()

    def test_base_url_trailing_slash(self):
        client = KomputerClient("http://localhost:8080/")
        assert client._api_client.configuration.host == "http://localhost:8080/api/v1"
        client.close()

    def test_context_manager(self):
        with KomputerClient("http://localhost:8080") as client:
            assert isinstance(client.agents, AgentsApi)

    def test_default_base_url(self):
        client = KomputerClient()
        assert client._api_client.configuration.host == "http://localhost:8080/api/v1"
        client.close()


class TestKwargsMethodsExist:
    """Verify all kwargs convenience methods exist on KomputerClient."""

    def setup_method(self):
        self.client = KomputerClient("http://localhost:8080")

    def teardown_method(self):
        self.client.close()

    # Agents
    def test_has_create_agent(self):
        assert callable(self.client.create_agent)

    def test_has_list_agents(self):
        assert callable(self.client.list_agents)

    def test_has_get_agent(self):
        assert callable(self.client.get_agent)

    def test_has_patch_agent(self):
        assert callable(self.client.patch_agent)

    def test_has_delete_agent(self):
        assert callable(self.client.delete_agent)

    def test_has_cancel_agent(self):
        assert callable(self.client.cancel_agent_task)

    def test_has_get_agent_events(self):
        assert callable(self.client.get_agent_events)

    def test_has_watch_agent(self):
        assert callable(self.client.watch_agent)

    # Memories
    def test_has_create_memory(self):
        assert callable(self.client.create_memory)

    def test_has_list_memories(self):
        assert callable(self.client.list_memories)

    def test_has_get_memory(self):
        assert callable(self.client.get_memory)

    def test_has_patch_memory(self):
        assert callable(self.client.patch_memory)

    def test_has_delete_memory(self):
        assert callable(self.client.delete_memory)

    # Skills
    def test_has_create_skill(self):
        assert callable(self.client.create_skill)

    def test_has_list_skills(self):
        assert callable(self.client.list_skills)

    def test_has_get_skill(self):
        assert callable(self.client.get_skill)

    def test_has_patch_skill(self):
        assert callable(self.client.patch_skill)

    def test_has_delete_skill(self):
        assert callable(self.client.delete_skill)

    # Schedules
    def test_has_create_schedule(self):
        assert callable(self.client.create_schedule)

    def test_has_list_schedules(self):
        assert callable(self.client.list_schedules)

    def test_has_get_schedule(self):
        assert callable(self.client.get_schedule)

    def test_has_patch_schedule(self):
        assert callable(self.client.patch_schedule)

    def test_has_delete_schedule(self):
        assert callable(self.client.delete_schedule)

    # Secrets
    def test_has_create_secret(self):
        assert callable(self.client.create_secret)

    def test_has_list_secrets(self):
        assert callable(self.client.list_secrets)

    def test_has_update_secret(self):
        assert callable(self.client.update_secret)

    def test_has_delete_secret(self):
        assert callable(self.client.delete_secret)

    # Connectors
    def test_has_create_connector(self):
        assert callable(self.client.create_connector)

    def test_has_list_connectors(self):
        assert callable(self.client.list_connectors)

    def test_has_get_connector(self):
        assert callable(self.client.get_connector)

    def test_has_delete_connector(self):
        assert callable(self.client.delete_connector)

    # Offices
    def test_has_list_offices(self):
        assert callable(self.client.list_offices)

    def test_has_get_office(self):
        assert callable(self.client.get_office)

    def test_has_delete_office(self):
        assert callable(self.client.delete_office)
