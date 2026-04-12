#!/usr/bin/env python3
"""Generate kwargs-style client wrapper from the OpenAPI spec.

Reads openapi.yaml and generates komputer_ai/client.py with convenience
methods that accept keyword arguments instead of model objects.

Usage:
    cd komputer-sdk
    python generate_client.py
"""

import re
import yaml
from pathlib import Path

SPEC_PATH = Path(__file__).parent / "openapi.yaml"
OUTPUT_PATH = Path(__file__).parent / "python" / "komputer_ai" / "client.py"

# Map OpenAPI tags to the generated API class names and attribute names
TAG_MAP = {
    "agents": ("AgentsApi", "agents"),
    "offices": ("OfficesApi", "offices"),
    "schedules": ("SchedulesApi", "schedules"),
    "memories": ("MemoriesApi", "memories"),
    "skills": ("SkillsApi", "skills"),
    "secrets": ("SecretsApi", "secrets"),
    "connectors": ("ConnectorsApi", "connectors"),
    "templates": ("TemplatesApi", "templates"),
}

# Operations to skip (websocket, internal, etc.)
SKIP_OPERATIONS = {"agentsNameWsGet", "namespacesGet"}

# Type mapping from OpenAPI to Python
TYPE_MAP = {
    "string": "str",
    "integer": "int",
    "boolean": "bool",
    "number": "float",
}


def to_snake_case(name):
    s1 = re.sub("(.)([A-Z][a-z]+)", r"\1_\2", name)
    return re.sub("([a-z0-9])([A-Z])", r"\1_\2", s1).lower()


def resolve_ref(spec, ref):
    parts = ref.lstrip("#/").split("/")
    obj = spec
    for p in parts:
        obj = obj[p]
    return obj


def get_python_type(prop, spec):
    if "$ref" in prop:
        schema = resolve_ref(spec, prop["$ref"])
        return get_python_type(schema, spec)
    t = prop.get("type", "object")
    if t == "array":
        items = prop.get("items", {})
        item_type = TYPE_MAP.get(items.get("type", "string"), "str")
        return f"List[{item_type}]"
    if t == "object" and "additionalProperties" in prop:
        val_type = TYPE_MAP.get(prop["additionalProperties"].get("type", "string"), "str")
        return f"Dict[str, {val_type}]"
    return TYPE_MAP.get(t, "str")


def get_model_class_name(ref):
    """Extract class name from $ref like '#/components/schemas/AgentResponse'."""
    return ref.split("/")[-1]


def get_request_body_fields(spec, operation):
    """Extract fields from the request body schema."""
    body = operation.get("requestBody", {})
    content = body.get("content", {})
    json_content = content.get("application/json", {})
    schema = json_content.get("schema", {})

    if "$ref" in schema:
        schema = resolve_ref(spec, schema["$ref"])

    fields = []
    required_fields = set(schema.get("required", []))
    properties = schema.get("properties", {})

    for prop_name, prop_schema in properties.items():
        python_name = to_snake_case(prop_name)
        python_type = get_python_type(prop_schema, spec)
        required = prop_name in required_fields
        fields.append({
            "json_name": prop_name,
            "python_name": python_name,
            "python_type": python_type,
            "required": required,
            "description": prop_schema.get("description", ""),
        })

    return fields


def get_path_params(operation):
    """Extract path parameters."""
    params = []
    for p in operation.get("parameters", []):
        if p.get("in") == "path":
            params.append({
                "json_name": p["name"],
                "python_name": to_snake_case(p["name"]),
                "python_type": TYPE_MAP.get(p.get("schema", {}).get("type", "string"), "str"),
                "required": True,
            })
    return params


def get_model_name_for_body(spec, operation):
    """Get the model class name for the request body, if any."""
    body = operation.get("requestBody", {})
    content = body.get("content", {})
    json_content = content.get("application/json", {})
    schema = json_content.get("schema", {})
    if "$ref" in schema:
        return get_model_class_name(schema["$ref"])
    return None


def generate_method(spec, operation_id, method, path, operation):
    """Generate a single convenience method."""
    path_params = get_path_params(operation)
    body_fields = get_request_body_fields(spec, operation)
    model_name = get_model_name_for_body(spec, operation)
    tag = operation.get("tags", [""])[0]
    tag_info = TAG_MAP.get(tag)
    if not tag_info:
        return None, None

    api_attr = tag_info[1]
    method_name = to_snake_case(operation_id)

    # Build the generated API method name (what openapi-generator creates)
    api_method_name = method_name

    # Build signature
    required_args = []
    optional_args = []

    for p in path_params:
        required_args.append(f"{p['python_name']}: {p['python_type']}")

    # Sort required fields: 'name' first, then others alphabetically
    required_body = sorted(
        [f for f in body_fields if f["required"]],
        key=lambda f: (0 if f["python_name"] == "name" else 1, f["python_name"]),
    )
    optional_body = sorted(
        [f for f in body_fields if not f["required"]],
        key=lambda f: f["python_name"],
    )

    for f in required_body:
        required_args.append(f"{f['python_name']}: {f['python_type']}")

    for f in optional_body:
        optional_args.append(f"{f['python_name']}: Optional[{f['python_type']}] = None")

    # Build the method
    all_args = required_args.copy()
    if optional_args:
        if all_args:
            all_args.append("*")
        all_args.extend(optional_args)

    sig_args = ", ".join(["self"] + all_args)

    # Build the body call
    if model_name and body_fields:
        model_args = ", ".join(
            f"{f['python_name']}={f['python_name']}" for f in body_fields
        )
        path_arg_values = ", ".join(p["python_name"] for p in path_params)
        if path_params:
            call = f"return self.{api_attr}.{api_method_name}({path_arg_values}, {model_name}({model_args}))"
        else:
            call = f"return self.{api_attr}.{api_method_name}({model_name}({model_args}))"
    else:
        arg_values = ", ".join(p["python_name"] for p in path_params)
        call = f"return self.{api_attr}.{api_method_name}({arg_values})"

    return method_name, f"    def {method_name}({sig_args}):\n        {call}\n", model_name


def main():
    with open(SPEC_PATH) as f:
        spec = yaml.safe_load(f)

    methods_by_tag = {}
    model_imports = set()

    paths = spec.get("paths", {})
    for path, path_item in paths.items():
        for http_method in ["get", "post", "put", "patch", "delete"]:
            operation = path_item.get(http_method)
            if not operation:
                continue

            operation_id = operation.get("operationId", "")
            if not operation_id or operation_id in SKIP_OPERATIONS:
                continue

            tag = operation.get("tags", [""])[0]
            if tag not in TAG_MAP:
                continue

            method_name, method_code, model_name = generate_method(
                spec, operation_id, http_method, path, operation
            )
            if method_code:
                methods_by_tag.setdefault(tag, []).append(method_code)
                if model_name:
                    model_imports.add(model_name)

    # Generate the file
    api_imports = []
    for tag, (class_name, _) in sorted(TAG_MAP.items()):
        module_name = to_snake_case(class_name).replace("_api", "_api")
        api_imports.append(f"from komputer_ai.api.{module_name} import {class_name}")

    model_import_list = ", ".join(sorted(model_imports))

    sections = []
    tag_order = ["agents", "memories", "skills", "schedules", "secrets", "connectors", "offices", "templates"]
    for tag in tag_order:
        if tag in methods_by_tag:
            section_title = tag.capitalize()
            section = f"    # --- {section_title} ---\n\n"
            section += "\n".join(methods_by_tag[tag])
            sections.append(section)

    methods_block = "\n".join(sections)

    output = f'''"""High-level convenience client for the komputer.ai API.

Auto-generated by generate_client.py — do not edit manually.

Quick start:
    client = KomputerClient("http://localhost:8080")
    client.create_agent(name="my-agent", instructions="Say hello", model="claude-sonnet-4-6")

    for event in client.watch_agent("my-agent"):
        print(event.type, event.payload)

Direct API access (model-based):
    from komputer_ai.models import CreateAgentRequest
    client.agents.create_agent(CreateAgentRequest(name="my-agent", instructions="..."))
"""

from typing import Dict, List, Optional

from komputer_ai import Configuration, ApiClient
{chr(10).join(api_imports)}
from komputer_ai.api.agents_ws import AgentEventStream
from komputer_ai.models import (
    {model_import_list},
)


class KomputerClient:
    """Client for the komputer.ai API.

    Provides kwargs-style methods for common operations and direct access
    to the generated API clients via .agents, .memories, .skills, etc.
    """

    def __init__(self, base_url: str = "http://localhost:8080"):
        self._base_url = base_url.rstrip("/")
        config = Configuration(host=f"{{self._base_url}}/api/v1")
        api_client = ApiClient(config)

{chr(10).join(f"        self.{attr} = {cls}(api_client)" for _, (cls, attr) in sorted(TAG_MAP.items()))}
        self._api_client = api_client

{methods_block}

    # --- WebSocket ---

    def watch_agent(self, name: str) -> AgentEventStream:
        """Stream real-time events from an agent via WebSocket.

        Requires: pip install websocket-client
        """
        ws_url = self._base_url.replace("http://", "ws://").replace(
            "https://", "wss://"
        )
        return AgentEventStream(ws_url, name)

    # --- Lifecycle ---

    def close(self):
        self._api_client.__exit__(None, None, None)

    def __enter__(self):
        return self

    def __exit__(self, *args):
        self._api_client.__exit__(*args)
'''

    OUTPUT_PATH.write_text(output)
    print(f"Generated {OUTPUT_PATH}")
    print(f"  {sum(len(m) for m in methods_by_tag.values())} methods across {len(methods_by_tag)} resource groups")
    print(f"  {len(model_imports)} model imports")


if __name__ == "__main__":
    main()
