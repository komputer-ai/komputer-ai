"""OAuth token refresh helper — calls komputer-api to refresh expired tokens."""

import logging
import os
import httpx

logger = logging.getLogger(__name__)

API_URL = os.environ.get("KOMPUTER_API_URL", "http://komputer-api:8080")
NAMESPACE = os.environ.get("KOMPUTER_NAMESPACE", "default")


async def refresh_oauth_token(connector_name: str) -> str | None:
    """Call komputer-api to refresh an OAuth token. Returns new access_token or None."""
    try:
        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.post(
                f"{API_URL}/api/v1/oauth/refresh",
                json={"connector_name": connector_name, "namespace": NAMESPACE},
            )
            if resp.status_code == 200:
                data = resp.json()
                return data.get("access_token")
    except Exception as e:
        logger.exception("oauth refresh failed", extra={"connector_name": connector_name, "error": str(e)})
    return None
