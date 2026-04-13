#!/usr/bin/env python3
"""Generate convenience client wrappers from the OpenAPI spec.

Reads openapi.yaml and generates language-specific client wrappers that
accept keyword arguments / flat parameters instead of model objects.

Usage:
    cd komputer-sdk
    python generate_client.py              # generate all languages
    python generate_client.py python       # generate python only
    python generate_client.py go           # generate go only
    python generate_client.py typescript   # generate typescript only
"""

import re
import sys
import yaml
from pathlib import Path

SPEC_PATH = Path(__file__).parent / "openapi.yaml"

# Map OpenAPI tags to API class/attribute names (shared across languages)
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

TAG_ORDER = ["agents", "memories", "skills", "schedules", "secrets", "connectors", "offices", "templates"]

SKIP_OPERATIONS = {"agentsNameWsGet", "namespacesGet"}


# --- Shared helpers ---

def to_snake_case(name):
    s1 = re.sub("(.)([A-Z][a-z]+)", r"\1_\2", name)
    return re.sub("([a-z0-9])([A-Z])", r"\1_\2", s1).lower()


def to_pascal_case(name):
    return re.sub(r"(?:^|_)([a-z])", lambda m: m.group(1).upper(), name)


def to_camel_case(name):
    pascal = to_pascal_case(name)
    return pascal[0].lower() + pascal[1:] if pascal else ""


def resolve_ref(spec, ref):
    parts = ref.lstrip("#/").split("/")
    obj = spec
    for p in parts:
        obj = obj[p]
    return obj


def get_model_class_name(ref):
    return ref.split("/")[-1]


def get_openapi_type(prop, spec):
    if "$ref" in prop:
        schema = resolve_ref(spec, prop["$ref"])
        return get_openapi_type(schema, spec)
    return prop.get("type", "object"), prop


def get_request_body_fields(spec, operation):
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
        oa_type, full_schema = get_openapi_type(prop_schema, spec)
        required = prop_name in required_fields
        # Track if this is a $ref (nested object type)
        ref_name = None
        if "$ref" in prop_schema:
            ref_name = get_model_class_name(prop_schema["$ref"])
        fields.append({
            "json_name": prop_name,
            "oa_type": oa_type,
            "oa_schema": full_schema,
            "required": required,
            "description": prop_schema.get("description", ""),
            "ref_name": ref_name,
        })

    return fields


def get_path_params(operation):
    params = []
    for p in operation.get("parameters", []):
        if p.get("in") == "path":
            params.append({
                "json_name": p["name"],
                "oa_type": p.get("schema", {}).get("type", "string"),
                "required": True,
            })
    return params


def get_model_name_for_body(spec, operation):
    body = operation.get("requestBody", {})
    content = body.get("content", {})
    json_content = content.get("application/json", {})
    schema = json_content.get("schema", {})
    if "$ref" in schema:
        return get_model_class_name(schema["$ref"])
    return None


def parse_operations(spec):
    """Parse all operations from the spec into a structured list."""
    operations = []
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
            operations.append({
                "operation_id": operation_id,
                "http_method": http_method,
                "path": path,
                "tag": tag,
                "path_params": get_path_params(operation),
                "body_fields": get_request_body_fields(spec, operation),
                "model_name": get_model_name_for_body(spec, operation),
                "operation": operation,
            })
    return operations


def sort_body_fields(fields):
    """Sort body fields: 'name' first among required, then alphabetical."""
    required = sorted(
        [f for f in fields if f["required"]],
        key=lambda f: (0 if f["json_name"] == "name" else 1, f["json_name"]),
    )
    optional = sorted(
        [f for f in fields if not f["required"]],
        key=lambda f: f["json_name"],
    )
    return required, optional


# --- Python generator ---

PYTHON_TYPE_MAP = {"string": "str", "integer": "int", "boolean": "bool", "number": "float"}


def python_type(field):
    if field.get("ref_name"):
        return field["ref_name"]
    t = field["oa_type"]
    schema = field.get("oa_schema", {})
    if t == "array":
        items = schema.get("items", {})
        item_type = PYTHON_TYPE_MAP.get(items.get("type", "string"), "str")
        return f"List[{item_type}]"
    if t == "object" and "additionalProperties" in schema:
        val_type = PYTHON_TYPE_MAP.get(schema["additionalProperties"].get("type", "string"), "str")
        return f"Dict[str, {val_type}]"
    return PYTHON_TYPE_MAP.get(t, "str")


def generate_python(operations):
    output_path = Path(__file__).parent / "python" / "komputer_ai" / "client.py"
    methods_by_tag = {}
    model_imports = set()

    for op in operations:
        tag = op["tag"]
        api_attr = TAG_MAP[tag][1]
        method_name = to_snake_case(op["operation_id"])
        required_body, optional_body = sort_body_fields(op["body_fields"])

        # Build signature
        required_args = []
        optional_args = []
        for p in op["path_params"]:
            required_args.append(f"{to_snake_case(p['json_name'])}: {PYTHON_TYPE_MAP.get(p['oa_type'], 'str')}")
        for f in required_body:
            required_args.append(f"{to_snake_case(f['json_name'])}: {python_type(f)}")
        for f in optional_body:
            optional_args.append(f"{to_snake_case(f['json_name'])}: Optional[{python_type(f)}] = None")

        all_args = required_args.copy()
        if optional_args:
            if all_args:
                all_args.append("*")
            all_args.extend(optional_args)
        sig_args = ", ".join(["self"] + all_args)

        # Collect ref type imports from fields
        for f in op["body_fields"]:
            if f.get("ref_name"):
                model_imports.add(f["ref_name"])

        # Build call
        if op["model_name"] and op["body_fields"]:
            model_imports.add(op["model_name"])
            model_args = ", ".join(f"{to_snake_case(f['json_name'])}={to_snake_case(f['json_name'])}" for f in op["body_fields"])
            path_args = ", ".join(to_snake_case(p["json_name"]) for p in op["path_params"])
            if path_args:
                call = f"return self.{api_attr}.{method_name}({path_args}, {op['model_name']}({model_args}))"
            else:
                call = f"return self.{api_attr}.{method_name}({op['model_name']}({model_args}))"
        else:
            args = ", ".join(to_snake_case(p["json_name"]) for p in op["path_params"])
            call = f"return self.{api_attr}.{method_name}({args})"

        code = f"    def {method_name}({sig_args}):\n        {call}\n"
        methods_by_tag.setdefault(tag, []).append(code)

    # Render
    api_imports = []
    for tag, (class_name, _) in sorted(TAG_MAP.items()):
        module_name = to_snake_case(class_name)
        api_imports.append(f"from komputer_ai.api.{module_name} import {class_name}")

    model_import_list = ", ".join(sorted(model_imports))
    sections = []
    for tag in TAG_ORDER:
        if tag in methods_by_tag:
            sections.append(f"    # --- {tag.capitalize()} ---\n\n" + "\n".join(methods_by_tag[tag]))

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
from komputer_ai.api.agents_ws import AgentEvent, AgentEventStream, Payload
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

{chr(10).join(sections)}

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
    output_path.write_text(output)
    print(f"  Python: {output_path} ({sum(len(m) for m in methods_by_tag.values())} methods)")


# --- Go generator ---

GO_TYPE_MAP = {"string": "string", "integer": "int64", "boolean": "bool", "number": "float64"}


def go_type(field, pointer=False):
    if field.get("ref_name"):
        ref = f"komputer.{field['ref_name']}"
        return f"*{ref}" if pointer else ref
    t = field["oa_type"]
    schema = field.get("oa_schema", {})
    if t == "array":
        items = schema.get("items", {})
        item_type = GO_TYPE_MAP.get(items.get("type", "string"), "string")
        return f"[]{item_type}"
    if t == "object" and "additionalProperties" in schema:
        val_type = GO_TYPE_MAP.get(schema["additionalProperties"].get("type", "string"), "string")
        return f"map[string]{val_type}"
    base = GO_TYPE_MAP.get(t, "string")
    if pointer:
        return f"*{base}"
    return base


def generate_go(operations):
    output_path = Path(__file__).parent / "go" / "client.go"
    methods_by_tag = {}

    for op in operations:
        tag = op["tag"]
        method_name = to_pascal_case(to_snake_case(op["operation_id"]))
        required_body, optional_body = sort_body_fields(op["body_fields"])

        # Build func params
        params = ["ctx context.Context"]
        for p in op["path_params"]:
            params.append(f"{to_camel_case(to_snake_case(p['json_name']))} {GO_TYPE_MAP.get(p['oa_type'], 'string')}")
        for f in required_body:
            params.append(f"{to_camel_case(to_snake_case(f['json_name']))} {go_type(f)}")

        # Options struct fields
        opts_fields = []
        for f in optional_body:
            field_name = to_pascal_case(to_snake_case(f["json_name"]))
            opts_fields.append(f"\t{field_name} {go_type(f, pointer=True)}")

        has_opts = len(opts_fields) > 0
        opts_struct_name = f"{method_name}Opts"

        if has_opts:
            params.append(f"opts ...{opts_struct_name}")

        sig = ", ".join(params)

        # Build the model struct literal
        model_name = op.get("model_name")
        if model_name and op["body_fields"]:
            struct_lines = []
            for f in op["body_fields"]:
                go_field = to_pascal_case(to_snake_case(f["json_name"]))
                go_var = to_camel_case(to_snake_case(f["json_name"]))
                if f["required"]:
                    struct_lines.append(f"\t\t{go_field}: {go_var},")
                else:
                    # Set from opts if provided
                    pass  # handled below

            # Build the method body
            body_lines = []
            body_lines.append(f"\treq := komputer.{model_name}{{")
            for f in required_body:
                go_field = to_pascal_case(to_snake_case(f["json_name"]))
                go_var = to_camel_case(to_snake_case(f["json_name"]))
                body_lines.append(f"\t\t{go_field}: {go_var},")
            body_lines.append("\t}")

            if has_opts:
                body_lines.append("\tif len(opts) > 0 {")
                body_lines.append("\t\to := opts[0]")
                for f in optional_body:
                    go_field = to_pascal_case(to_snake_case(f["json_name"]))
                    body_lines.append(f"\t\tif o.{go_field} != nil {{")
                    body_lines.append(f"\t\t\treq.{go_field} = o.{go_field}")
                    body_lines.append("\t\t}")
                body_lines.append("\t}")

            # Build the API call
            path_args = ", ".join(to_camel_case(to_snake_case(p["json_name"])) for p in op["path_params"])
            api_service = to_pascal_case(tag) + "API"
            if path_args:
                body_lines.append(f"\treturn c.api.{api_service}.{method_name}(ctx, {path_args}).Request(req).Execute()")
            else:
                body_lines.append(f"\treturn c.api.{api_service}.{method_name}(ctx).Request(req).Execute()")
        else:
            body_lines = []
            path_args = ", ".join(to_camel_case(to_snake_case(p["json_name"])) for p in op["path_params"])
            api_service = to_pascal_case(tag) + "API"
            if path_args:
                body_lines.append(f"\treturn c.api.{api_service}.{method_name}(ctx, {path_args}).Execute()")
            else:
                body_lines.append(f"\treturn c.api.{api_service}.{method_name}(ctx).Execute()")

        # Determine return type
        resp_ref = None
        resp_map_type = None
        responses = op["operation"].get("responses", {})
        for code in ["200", "201"]:
            resp = responses.get(code, {})
            content = resp.get("content", {}).get("application/json", {}).get("schema", {})
            if "$ref" in content:
                resp_ref = get_model_class_name(content["$ref"])
                break
            elif content.get("type") == "object":
                addl = content.get("additionalProperties", {})
                if isinstance(addl, dict) and addl.get("type") == "string":
                    resp_map_type = "map[string]string"
                else:
                    resp_map_type = "map[string]interface{}"
                break

        if resp_ref:
            return_type = f"(*komputer.{resp_ref}, *http.Response, error)"
        elif resp_map_type:
            return_type = f"({resp_map_type}, *http.Response, error)"
        else:
            return_type = "(*http.Response, error)"

        # Assemble
        method_code = f"func (c *Client) {method_name}({sig}) {return_type} {{\n"
        method_code += "\n".join(body_lines)
        method_code += "\n}\n"

        if has_opts:
            opts_code = f"type {opts_struct_name} struct {{\n"
            opts_code += "\n".join(opts_fields)
            opts_code += "\n}\n"
            methods_by_tag.setdefault(tag, []).append(opts_code)

        methods_by_tag.setdefault(tag, []).append(method_code)

    # Render
    sections = []
    for tag in TAG_ORDER:
        if tag in methods_by_tag:
            sections.append(f"// --- {tag.capitalize()} ---\n\n" + "\n".join(methods_by_tag[tag]))

    output = f'''// Package client provides a convenience wrapper for the komputer.ai API.
//
// Auto-generated by generate_client.py — do not edit manually.
//
// Quick start:
//
//	client := client.New("http://localhost:8080")
//	agent, _, err := client.CreateAgent(ctx, "my-agent", "Say hello",
//	    client.CreateAgentOpts{{Model: client.PtrString("claude-sonnet-4-6")}})
package client

import (
\t"context"
\t"net/http"

\tkomputer "github.com/kontroloop-ai/komputer-ai/komputer-sdk/go/internal"
)

// Client wraps the generated komputer API client with convenience methods.
type Client struct {{
\tapi     *komputer.APIClient
\tbaseURL string
}}

// New creates a new Client for the given base URL.
func New(baseURL string) *Client {{
\tcfg := komputer.NewConfiguration()
\tcfg.Servers = komputer.ServerConfigurations{{
\t\t{{URL: baseURL + "/api/v1"}},
\t}}
\treturn &Client{{api: komputer.NewAPIClient(cfg), baseURL: baseURL}}
}}

{chr(10).join(sections)}
'''
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(output)
    print(f"  Go:     {output_path} ({sum(len(m) for m in methods_by_tag.values())} methods/types)")


# --- TypeScript generator ---

TS_TYPE_MAP = {"string": "string", "integer": "number", "boolean": "boolean", "number": "number"}


def ts_type(field):
    if field.get("ref_name"):
        return field["ref_name"]
    t = field["oa_type"]
    schema = field.get("oa_schema", {})
    if t == "array":
        items = schema.get("items", {})
        item_type = TS_TYPE_MAP.get(items.get("type", "string"), "string")
        return f"{item_type}[]"
    if t == "object" and "additionalProperties" in schema:
        val_type = TS_TYPE_MAP.get(schema["additionalProperties"].get("type", "string"), "string")
        return f"Record<string, {val_type}>"
    return TS_TYPE_MAP.get(t, "string")


def generate_typescript(operations):
    output_path = Path(__file__).parent / "typescript" / "src" / "client.ts"
    methods_by_tag = {}
    model_imports = set()
    api_imports = set()

    for op in operations:
        tag = op["tag"]
        method_name = to_camel_case(to_snake_case(op["operation_id"]))
        required_body, optional_body = sort_body_fields(op["body_fields"])
        api_class = to_pascal_case(tag) + "Api"
        api_imports.add(api_class)

        # Build params interface fields
        all_fields = []
        for p in op["path_params"]:
            all_fields.append((to_camel_case(to_snake_case(p["json_name"])), TS_TYPE_MAP.get(p["oa_type"], "string"), True))
        for f in required_body:
            all_fields.append((to_camel_case(to_snake_case(f["json_name"])), ts_type(f), True))
        for f in optional_body:
            all_fields.append((to_camel_case(to_snake_case(f["json_name"])), ts_type(f), False))

        model_name = op.get("model_name")
        if model_name:
            model_imports.add(model_name)
        for f in op["body_fields"]:
            if f.get("ref_name"):
                model_imports.add(f["ref_name"])

        # For methods with no params or only path params with no body
        has_params = len(all_fields) > 0

        if not has_params:
            # Simple method, no params
            method_code = f"  async {method_name}() {{\n"
            method_code += f"    return this._{tag}.{method_name}({{}});\n"
            method_code += "  }\n"
        elif not op["body_fields"]:
            # Only path params — pass directly as the operation request
            param_fields = ", ".join(f"{name}: {typ}" for name, typ, _ in all_fields)
            request_fields = ", ".join(f"{name}" for name, _, _ in all_fields)
            method_code = f"  async {method_name}({param_fields}) {{\n"
            method_code += f"    return this._{tag}.{method_name}({{ {request_fields} }});\n"
            method_code += "  }\n"
        else:
            # Has body fields — need to separate path params from body
            path_param_names = {to_camel_case(to_snake_case(p["json_name"])) for p in op["path_params"]}

            # Build the params type inline
            param_parts = []
            for name, typ, req in all_fields:
                opt = "" if req else "?"
                param_parts.append(f"{name}{opt}: {typ}")

            params_str = "params: { " + "; ".join(param_parts) + " }"
            method_code = f"  async {method_name}({params_str}) {{\n"

            # Build the operation request object
            # Path params go at top level, body fields go in 'request'
            op_parts = []
            for p in op["path_params"]:
                pn = to_camel_case(to_snake_case(p["json_name"]))
                op_parts.append(f"{pn}: params.{pn}")

            body_parts = []
            for f in op["body_fields"]:
                fn = to_camel_case(to_snake_case(f["json_name"]))
                body_parts.append(f"{fn}: params.{fn}")

            body_obj = "{ " + ", ".join(body_parts) + " }"
            if op_parts:
                all_op_parts = ", ".join(op_parts) + f", request: {body_obj}"
            else:
                all_op_parts = f"request: {body_obj}"

            method_code += f"    return this._{tag}.{method_name}({{ {all_op_parts} }});\n"
            method_code += "  }\n"

        methods_by_tag.setdefault(tag, []).append(method_code)

    # Render
    sections = []
    for tag in TAG_ORDER:
        if tag in methods_by_tag:
            sections.append(f"  // --- {tag.capitalize()} ---\n\n" + "\n".join(methods_by_tag[tag]))

    api_import_list = ", ".join(sorted(api_imports))
    model_import_list = ", ".join(sorted(model_imports))

    # Build constructor assignments
    constructor_lines = []
    for tag in sorted(TAG_MAP.keys()):
        api_class = to_pascal_case(tag) + "Api"
        constructor_lines.append(f"    this._{tag} = new {api_class}(config);")

    output = f'''/**
 * High-level convenience client for the komputer.ai API.
 *
 * Auto-generated by generate_client.py — do not edit manually.
 *
 * @example
 * const client = new KomputerClient("http://localhost:8080");
 * const agent = await client.createAgent({{
 *   name: "my-agent",
 *   instructions: "Say hello",
 *   model: "claude-sonnet-4-6",
 * }});
 */

import {{ Configuration }} from "./runtime";
import {{ {api_import_list} }} from "./apis";
import type {{ {model_import_list} }} from "./models";
import {{ AgentEventStream }} from "./watch";
export type {{ AgentEvent }} from "./watch";

export class KomputerClient {{
{chr(10).join(f"  private _{tag}: {to_pascal_case(tag)}Api;" for tag in sorted(TAG_MAP.keys()))}
  private _baseUrl: string;

  constructor(baseUrl: string = "http://localhost:8080") {{
    this._baseUrl = baseUrl.replace(/\\/$/, "");
    const config = new Configuration({{ basePath: this._baseUrl + "/api/v1" }});
{chr(10).join(constructor_lines)}
  }}

{chr(10).join(sections)}

  // --- WebSocket ---

  watchAgent(name: string): AgentEventStream {{
    const wsUrl = this._baseUrl.replace("http://", "ws://").replace("https://", "wss://");
    return new AgentEventStream(wsUrl, name);
  }}
}}
'''
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(output)
    print(f"  TS:     {output_path} ({sum(len(m) for m in methods_by_tag.values())} methods)")


# --- Main ---

def main():
    with open(SPEC_PATH) as f:
        spec = yaml.safe_load(f)

    operations = parse_operations(spec)
    targets = sys.argv[1:] if len(sys.argv) > 1 else ["python", "go", "typescript"]

    print(f"Generating client wrappers ({len(operations)} operations)...")

    if "python" in targets:
        generate_python(operations)
    if "go" in targets:
        generate_go(operations)
    if "typescript" in targets:
        generate_typescript(operations)

    print("Done.")


if __name__ == "__main__":
    main()
