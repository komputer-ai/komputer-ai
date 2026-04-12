# komputer-sdk generation pipeline
# Usage:
#   make python     — regenerate Python SDK
#   make all        — regenerate all SDKs
#   make spec       — just regenerate + convert the OpenAPI spec

REPO_ROOT := $(shell git rev-parse --show-toplevel)
API_DIR := $(REPO_ROOT)/komputer-api
SPEC_SWAGGER := $(API_DIR)/docs/swagger.json
SPEC_OPENAPI := openapi.yaml
GENERATOR := npx @openapitools/openapi-generator-cli

.PHONY: all spec python client go typescript clean swagger test test-integration

all: python

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
	rm -rf python/komputer_ai python/docs python/test
	$(GENERATOR) generate \
		-i $(SPEC_OPENAPI) \
		-g python \
		-o python \
		--package-name komputer_ai \
		--skip-validate-spec \
		--additional-properties=projectName=komputer-ai,packageVersion=0.1.0
	python generate_client.py
	@echo "Python SDK generated in python/"

# Regenerate just the kwargs client wrapper (no spec/SDK regeneration)
client:
	python generate_client.py

# Placeholder targets for future SDKs
go: spec
	@echo "Go SDK generation not yet implemented"

typescript: spec
	@echo "TypeScript SDK generation not yet implemented"

# Run unit tests (no server needed)
test:
	cd python && pip install -e . -q && pytest tests/ --ignore=tests/integration -v

# Run integration tests (requires KOMPUTER_API_URL or localhost:8080)
test-integration:
	cd python && pip install -e . -q && pytest tests/integration -v

clean:
	rm -rf python/komputer_ai python/docs python/test python/*.egg-info
	rm -f $(SPEC_OPENAPI)
