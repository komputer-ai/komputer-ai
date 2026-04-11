"""Tests for the KomputerClient convenience wrapper and API class structure."""

from komputer_ai import Configuration, ApiClient
from komputer_ai.client import KomputerClient
from komputer_ai.api.agents_api import AgentsApi
from komputer_ai.api.offices_api import OfficesApi
from komputer_ai.api.schedules_api import SchedulesApi
from komputer_ai.api.memories_api import MemoriesApi
from komputer_ai.api.skills_api import SkillsApi
from komputer_ai.api.secrets_api import SecretsApi
from komputer_ai.api.connectors_api import ConnectorsApi
from komputer_ai.api.templates_api import TemplatesApi


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
        # Should not double-slash the URL
        assert client._api_client.configuration.host == "http://localhost:8080/api/v1"
        client.close()

    def test_context_manager(self):
        with KomputerClient("http://localhost:8080") as client:
            assert isinstance(client.agents, AgentsApi)

    def test_default_base_url(self):
        client = KomputerClient()
        assert client._api_client.configuration.host == "http://localhost:8080/api/v1"
        client.close()


class TestAgentsApiMethods:
    """Verify all expected CRUD methods exist on AgentsApi."""

    def setup_method(self):
        config = Configuration(host="http://localhost:8080/api/v1")
        self.api = AgentsApi(ApiClient(config))

    def test_has_create_agent(self):
        assert callable(self.api.create_agent)

    def test_has_list_agents(self):
        assert callable(self.api.list_agents)

    def test_has_get_agent(self):
        assert callable(self.api.get_agent)

    def test_has_delete_agent(self):
        assert callable(self.api.delete_agent)

    def test_has_patch_agent(self):
        assert callable(self.api.patch_agent)

    def test_has_cancel_agent_task(self):
        assert callable(self.api.cancel_agent_task)

    def test_has_get_agent_events(self):
        assert callable(self.api.get_agent_events)


class TestSchedulesApiMethods:
    def setup_method(self):
        config = Configuration(host="http://localhost:8080/api/v1")
        self.api = SchedulesApi(ApiClient(config))

    def test_has_create_schedule(self):
        assert callable(self.api.create_schedule)

    def test_has_list_schedules(self):
        assert callable(self.api.list_schedules)

    def test_has_get_schedule(self):
        assert callable(self.api.get_schedule)

    def test_has_delete_schedule(self):
        assert callable(self.api.delete_schedule)

    def test_has_patch_schedule(self):
        assert callable(self.api.patch_schedule)


class TestMemoriesApiMethods:
    def setup_method(self):
        config = Configuration(host="http://localhost:8080/api/v1")
        self.api = MemoriesApi(ApiClient(config))

    def test_has_create_memory(self):
        assert callable(self.api.create_memory)

    def test_has_list_memories(self):
        assert callable(self.api.list_memories)

    def test_has_get_memory(self):
        assert callable(self.api.get_memory)

    def test_has_delete_memory(self):
        assert callable(self.api.delete_memory)

    def test_has_patch_memory(self):
        assert callable(self.api.patch_memory)


class TestSkillsApiMethods:
    def setup_method(self):
        config = Configuration(host="http://localhost:8080/api/v1")
        self.api = SkillsApi(ApiClient(config))

    def test_has_create_skill(self):
        assert callable(self.api.create_skill)

    def test_has_list_skills(self):
        assert callable(self.api.list_skills)

    def test_has_get_skill(self):
        assert callable(self.api.get_skill)

    def test_has_delete_skill(self):
        assert callable(self.api.delete_skill)

    def test_has_patch_skill(self):
        assert callable(self.api.patch_skill)


class TestSecretsApiMethods:
    def setup_method(self):
        config = Configuration(host="http://localhost:8080/api/v1")
        self.api = SecretsApi(ApiClient(config))

    def test_has_create_secret(self):
        assert callable(self.api.create_secret)

    def test_has_list_secrets(self):
        assert callable(self.api.list_secrets)

    def test_has_delete_secret(self):
        assert callable(self.api.delete_secret)

    def test_has_update_secret(self):
        assert callable(self.api.update_secret)


class TestConnectorsApiMethods:
    def setup_method(self):
        config = Configuration(host="http://localhost:8080/api/v1")
        self.api = ConnectorsApi(ApiClient(config))

    def test_has_create_connector(self):
        assert callable(self.api.create_connector)

    def test_has_list_connectors(self):
        assert callable(self.api.list_connectors)

    def test_has_get_connector(self):
        assert callable(self.api.get_connector)

    def test_has_delete_connector(self):
        assert callable(self.api.delete_connector)

    def test_has_list_connector_tools(self):
        assert callable(self.api.list_connector_tools)


class TestOfficesApiMethods:
    def setup_method(self):
        config = Configuration(host="http://localhost:8080/api/v1")
        self.api = OfficesApi(ApiClient(config))

    def test_has_list_offices(self):
        assert callable(self.api.list_offices)

    def test_has_get_office(self):
        assert callable(self.api.get_office)

    def test_has_delete_office(self):
        assert callable(self.api.delete_office)

    def test_has_get_office_events(self):
        assert callable(self.api.get_office_events)
