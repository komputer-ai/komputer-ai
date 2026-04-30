"""
komputer.ai advanced Python SDK example.

Creates an agent using every field of KomputerClient.create_agent() — namespace,
template, model, role, lifecycle, system prompt, skills, memories, connectors,
secrets, and squad membership.

The full API reference lives in the live Swagger UI at /swagger/index.html on a
running komputer-api, and in komputer-sdk/openapi.yaml.

Requires: pip install komputer-ai-sdk
Pre-reqs in your namespace:
  - KomputerSkill: web-research, report-writing
  - KomputerMemory: team-glossary
  - KomputerConnector: slack-prod, jira-prod
  - K8s Secret: my-secret-key
  - KomputerAgentTemplate: research-template
Drop or replace any references your cluster doesn't have.
"""

from komputer_ai.client import KomputerClient
from komputer_ai.models import V1alpha1StorageSpec, V1PodSpec
from komputer_ai.models.v1_container import V1Container
from komputer_ai.models.v1_resource_requirements import V1ResourceRequirements

API_URL = "http://localhost:8080"


def run_advanced_agent() -> dict:
    with KomputerClient(API_URL) as client:
        agent = client.create_agent(
            # --- Required ---
            name="research-lead",
            instructions=(
                "Investigate the team's recent Slack discussions about cost overruns. "
                "Cross-reference any Jira tickets they mention. "
                "Produce a structured summary with action items and owners."
            ),

            # --- Where it runs ---
            namespace="research",                  # K8s namespace; defaults to server default
            template_ref="research-template",      # KomputerAgentTemplate or ClusterTemplate
            model="claude-sonnet-4-6",             # overrides the template's default model

            # --- Behavior ---
            role="manager",                        # "manager" gets sub-agent MCP tools; "" / None for plain worker
            lifecycle="Sleep",                     # "" (stay running), "Sleep" (pod deleted, PVC kept), "AutoDelete"
            system_prompt=(
                "Always cite sources with permalinks. "
                "Prefer concise structured summaries over prose. "
                "If a claim cannot be verified, mark it [unverified]."
            ),

            # --- Attached resources (must already exist in the namespace) ---
            skills=["web-research", "report-writing"],   # KomputerSkill names
            memories=["team-glossary"],                  # KomputerMemory names
            connectors=["slack-prod", "jira-prod"],      # KomputerConnector (MCP) names
            secret_refs=["my-secret-key"],              # K8s Secret names → mounted as SECRET_* env vars

            # --- Squad membership ---
            # Usually set automatically when a manager calls the create_agent MCP tool.
            # Pass it explicitly only if you're attaching this agent to an existing manager.
            office_manager=None,

            # --- Scheduling ---
            priority=10,                           # admission queue priority — higher gets in first

            # --- Workspace storage ---
            storage=V1alpha1StorageSpec(
                size="20Gi",                       # PVC size for /workspace; overrides template default
                # storage_class_name="fast-ssd",   # optional: override the cluster default storage class
            ),

            # --- Pod overrides (CPU/memory, node selector, etc.) ---
            # Merged on top of the template's pod spec. Use sparingly — usually
            # a KomputerAgentTemplate is the right place for cluster-wide pod tuning.
            pod_spec=V1PodSpec(
                containers=[
                    V1Container(
                        name="agent",              # must match the template's main container name
                        resources=V1ResourceRequirements(
                            requests={"cpu": "1", "memory": "2Gi"},
                            limits={"cpu": "4", "memory": "8Gi"},
                        ),
                    ),
                ],
                node_selector={"workload": "ai-agents"},
            ),
        )

        print(f"Agent created: {agent.name}")
        print(f"  namespace:   {agent.namespace}")
        print(f"  template:    {agent.template_ref}")
        print(f"  model:       {agent.model}")
        print(f"  role:        {agent.role}")
        print(f"  lifecycle:   {agent.lifecycle}")
        print(f"  skills:      {agent.skills}")
        print(f"  memories:    {agent.memories}")
        print(f"  connectors:  {agent.connectors}")
        print()

        for event in client.watch_agent(agent.name, namespace=agent.namespace):
            if event.type == "task_started":
                print(f"[started] {event.payload.get('instructions', '')[:80]}...")
            elif event.type == "thinking":
                print(f"[thinking] {event.payload.get('content', '')[:60]}...")
            elif event.type == "tool_call":
                print(f"[tool] {event.payload.get('tool')}: {str(event.payload.get('input', ''))[:60]}")
            elif event.type == "text":
                print(f"\n{event.payload.get('content', '')}")
            elif event.type == "task_completed":
                p = event.payload
                print(f"\n[done] cost=${p.get('cost_usd', 0):.4f}  "
                      f"duration={p.get('duration_ms', 0) / 1000:.1f}s  "
                      f"turns={p.get('turns', 0)}")
                return p
            elif event.type == "error":
                print(f"[error] {event.payload.get('error')}")
                break

    return {}


if __name__ == "__main__":
    run_advanced_agent()
