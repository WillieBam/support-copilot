"""HTTP API client for backend communications.

Provides a structured client interface (BackendClient) to communicate with the Go backend's
internal REST endpoints over the Docker network.
"""

import os
import logging
from typing import Optional
import httpx

BACKEND_BASE_URL = os.getenv("BACKEND_BASE_URL", "http://localhost:8080")
INTERNAL_API_KEY = os.getenv("INTERNAL_API_KEY", "dev-internal-key")

_logger = logging.getLogger("mcp2-client")

class BackendClient:
    def __init__(self):
        self._client = httpx.Client(
            base_url=BACKEND_BASE_URL,
            headers={"x-internal-api-key": INTERNAL_API_KEY},
            timeout=15.0,
        )

    # runbook methods
    def create_runbook(self, team_id: str, incident_id: str, title: str, content: str) -> dict:
        _logger.info("client=create_runbook team_id=%s incident_id=%s", team_id, incident_id)
        resp = self._client.post(
            f"/internal/teams/{team_id}/runbooks",
            json={"incident_id": incident_id, "title": title, "content": content},
        )
        resp.raise_for_status()
        return resp.json()

    def update_runbook(self, runbook_id: str, title: str, content: str) -> dict:
        _logger.info("client=update_runbook runbook_id=%s", runbook_id)
        resp = self._client.patch(
            f"/internal/runbooks/{runbook_id}",
            json={"title": title, "content": content},
        )
        resp.raise_for_status()
        return resp.json()

    def deprecate_runbook(self, runbook_id: str) -> dict:
        _logger.info("client=deprecate_runbook runbook_id=%s", runbook_id)
        resp = self._client.patch(f"/internal/runbooks/{runbook_id}/deprecate")
        resp.raise_for_status()
        return resp.json()

    def get_runbook(self, runbook_id: str) -> dict:
        _logger.info("client=get_runbook runbook_id=%s", runbook_id)
        resp = self._client.get(f"/internal/runbooks/{runbook_id}")
        resp.raise_for_status()
        return resp.json()

    def list_runbooks(self, team_id: str, status: Optional[str] = "active") -> list:
        _logger.info("client=list_runbooks team_id=%s status=%s", team_id, status)
        resp = self._client.get(
            f"/internal/teams/{team_id}/runbooks",
            params={"status": status or "active"},
        )
        resp.raise_for_status()
        return resp.json()

    # incident methods
    def get_incident_context(self, incident_id: str) -> dict:
        _logger.info("client=get_incident_context incident_id=%s", incident_id)
        resp = self._client.get(f"/internal/incidents/{incident_id}/context")
        resp.raise_for_status()
        return resp.json()

    def list_incidents(self, team_id: str) -> list:
        _logger.info("client=list_incidents team_id=%s", team_id)
        resp = self._client.get(f"/internal/teams/{team_id}/incidents")
        resp.raise_for_status()
        return resp.json()