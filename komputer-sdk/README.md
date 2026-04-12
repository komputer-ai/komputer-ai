# komputer-ai SDK

Auto-generated client libraries for the komputer.ai REST API.

## Python SDK

```bash
pip install komputer-ai    # or: cd komputer-sdk/python && pip install -e .
```

```python
from komputer_ai.client import KomputerClient

client = KomputerClient("http://localhost:8080")

# Create an agent
client.create_agent(
    name="my-agent",
    instructions="Analyze our Kubernetes cluster",
    model="claude-sonnet-4-6",
)

# Attach a memory
client.create_memory(name="context", content="We run a 50-node GKE cluster.")
client.patch_agent("my-agent", memories=["context"])

# Stream events as the agent works
for event in client.watch_agent("my-agent"):
    if event.type == "text":
        print(event.payload.get("content", ""))
    elif event.type == "task_completed":
        break

# List and clean up
agents = client.list_agents()
client.delete_agent("my-agent")
```

All methods accept keyword arguments directly — no model objects needed. For advanced use cases, the generated API clients are still available via `client.agents`, `client.memories`, etc.

## Regenerating

When the API changes, regenerate the SDKs:

```bash
cd komputer-sdk

# Regenerate everything (swagger → openapi → SDK + client wrapper)
make python

# Regenerate just the kwargs client wrapper
make client

# Just regenerate the OpenAPI spec
make spec
```

### Prerequisites

- [swag](https://github.com/swaggo/swag) — `go install github.com/swaggo/swag/cmd/swag@v1.16.6`
- [openapi-generator-cli](https://openapi-generator.tech/) — via `npx` (included in the Makefile)
- Node.js + npx (for openapi-generator-cli)
- Python 3.10+ (for `generate_client.py`)

## Testing

```bash
cd komputer-sdk

# Unit tests (no server needed)
make test

# Integration tests (requires a running komputer-ai instance)
KOMPUTER_API_URL=http://localhost:8080 make test-integration
```

Integration tests create and delete real resources (agents, memories, skills, etc.) prefixed with `sdk-test-`. They clean up after themselves.

## Structure

```
komputer-sdk/
├── Makefile              # Generation pipeline
├── generate_client.py    # Generates kwargs-style client wrapper from OpenAPI spec
├── openapi.yaml          # Generated OpenAPI 3.0 spec (committed for reference)
├── python/               # Python SDK
│   ├── komputer_ai/
│   │   ├── client.py     # Auto-generated kwargs convenience client
│   │   ├── api/          # Generated API classes (model-based)
│   │   │   └── agents_ws.py  # Hand-written WebSocket streaming
│   │   └── models/       # Generated request/response models
│   └── tests/
│       ├── test_client.py    # Client wrapper + API method tests
│       ├── test_models.py    # Model serialization tests
│       └── integration/      # Integration tests (requires live API)
├── go/                   # Go SDK (future)
└── typescript/           # TypeScript SDK (future)
```
