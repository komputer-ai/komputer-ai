# komputer-sdk generation pipeline
# Usage:
#   make python     — regenerate Python SDK
#   make all        — regenerate all SDKs
#   make spec       — just regenerate + convert the OpenAPI spec
#   make client     — regenerate client wrappers (overwrites hand-written files — use with care)

REPO_ROOT := $(shell git rev-parse --show-toplevel)
API_DIR := $(REPO_ROOT)/komputer-api
SPEC_SWAGGER := $(API_DIR)/docs/swagger.json
SPEC_OPENAPI := openapi.yaml
GENERATOR := npx @openapitools/openapi-generator-cli

.PHONY: all spec python client go typescript clean swagger test test-integration

all: python go typescript

# Step 1: Regenerate swagger from Go annotations
swagger:
	cd $(API_DIR) && swag init --parseDependency --parseInternal

# Step 2: Convert Swagger 2.0 → OpenAPI 3.0
spec: swagger
	$(GENERATOR) generate \
		-i $(SPEC_SWAGGER) \
		-g openapi-yaml \
		-o /tmp/openapi-convert \
		--skip-validate-spec
	cp /tmp/openapi-convert/openapi/openapi.yaml $(SPEC_OPENAPI)
	rm -rf /tmp/openapi-convert
	sed -i.bak 's/main\.//g' $(SPEC_OPENAPI) && rm -f $(SPEC_OPENAPI).bak
	@echo "OpenAPI 3.0 spec written to $(SPEC_OPENAPI)"

# Step 3: Generate Python SDK
python: spec
	$(GENERATOR) generate \
		-i $(SPEC_OPENAPI) \
		-g python \
		-o python \
		--package-name komputer_ai \
		--skip-validate-spec \
		--additional-properties=projectName=komputer-ai-sdk,packageVersion=0.1.0
	@echo "Python SDK generated in python/"

# Step 4: Generate Go SDK
go: spec
	$(GENERATOR) generate \
		-i $(SPEC_OPENAPI) \
		-g go \
		-o go/internal \
		--package-name komputer \
		--skip-validate-spec \
		--git-user-id komputer-ai \
		--git-repo-id komputer-ai \
		--additional-properties=packageVersion=0.1.0,isGoSubmodule=true
	rm -f go/internal/go.mod go/internal/go.sum go/internal/git_push.sh go/internal/.travis.yml
	rm -rf go/internal/.openapi-generator go/internal/test
	@echo "Go SDK generated in go/"

# Step 5: Generate TypeScript SDK
typescript: spec
	$(GENERATOR) generate \
		-i $(SPEC_OPENAPI) \
		-g typescript-fetch \
		-o typescript \
		--skip-validate-spec \
		--additional-properties=npmName=@komputer-ai/sdk,npmVersion=0.1.0,supportsES6=true
	@echo "TypeScript SDK generated in typescript/"

# Regenerate client wrappers from the OpenAPI spec.
# WARNING: This overwrites the hand-maintained client files (client.ts, client.py, client.go).
# Only use this to scaffold new methods after API changes, then manually merge.
client:
	python generate_client.py

# Run all unit tests
test: test-python test-go test-typescript

test-python:
	cd python && pip install -e . -q && pytest tests/ --ignore=tests/integration -v

test-go:
	cd go && go test -v ./...

test-typescript:
	cd typescript && npx vitest run

# Run integration tests (requires KOMPUTER_API_URL or localhost:8080)
test-integration:
	cd python && pip install -e . -q && pytest tests/integration -v

clean:
	rm -rf python/komputer_ai python/docs python/test python/*.egg-info
	rm -f $(SPEC_OPENAPI)
