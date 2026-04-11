# komputer-ai SDK

Auto-generated client libraries for the komputer.ai REST API.

## Regenerating

When the API changes, regenerate the SDKs:

```bash
cd komputer-sdk

# Regenerate everything (swagger → openapi → python SDK)
make python

# Or just regenerate the OpenAPI spec
make spec
```

### Prerequisites

- [swag](https://github.com/swaggo/swag) — `go install github.com/swaggo/swag/cmd/swag@v1.16.6`
- [openapi-generator-cli](https://openapi-generator.tech/) — via `npx` (included in the Makefile)
- Node.js + npx (for openapi-generator-cli)

### Python SDK

```bash
cd python
pip install -e .
```

```python
from komputer_ai.client import KomputerClient

with KomputerClient("http://localhost:8080") as client:
    agents = client.agents.list_agents()
    print(agents)
```

## Structure

```
komputer-sdk/
├── Makefile          # Generation pipeline
├── openapi.yaml      # Generated OpenAPI 3.0 spec (committed for reference)
├── python/           # Python SDK (auto-generated + client.py wrapper)
├── go/               # Go SDK (future)
└── typescript/       # TypeScript SDK (future)
```
