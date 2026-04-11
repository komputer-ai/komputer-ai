"""High-level convenience client for the komputer.ai API."""

from komputer_ai import Configuration, ApiClient
from komputer_ai.api.agents_api import AgentsApi
from komputer_ai.api.offices_api import OfficesApi
from komputer_ai.api.schedules_api import SchedulesApi
from komputer_ai.api.memories_api import MemoriesApi
from komputer_ai.api.skills_api import SkillsApi
from komputer_ai.api.secrets_api import SecretsApi
from komputer_ai.api.connectors_api import ConnectorsApi


class KomputerClient:
    """Convenience wrapper around the auto-generated komputer.ai API client.

    Usage:
        client = KomputerClient("http://localhost:8080")
        agents = client.agents.list_agents()
    """

    def __init__(self, base_url: str = "http://localhost:8080"):
        config = Configuration(host=f"{base_url.rstrip('/')}/api/v1")
        api_client = ApiClient(config)

        self.agents = AgentsApi(api_client)
        self.offices = OfficesApi(api_client)
        self.schedules = SchedulesApi(api_client)
        self.memories = MemoriesApi(api_client)
        self.skills = SkillsApi(api_client)
        self.secrets = SecretsApi(api_client)
        self.connectors = ConnectorsApi(api_client)
        self._api_client = api_client

    def close(self):
        self._api_client.__exit__(None, None, None)

    def __enter__(self):
        return self

    def __exit__(self, *args):
        self._api_client.__exit__(*args)
