"""MCP Server 2 — Knowledge Base & Incident Management.

Provides FastMCP tools to create, update, deprecate, and retrieve runbooks,
as well as fetching cleansed incident context for automated runbook generation.
"""

import logging
from typing import Optional
from fastmcp import FastMCP
from server_2.client import BackendClient

mcp = FastMCP("support-copilot-mcp-2")
_logger = logging.getLogger("mcp2-tools")
_client = BackendClient()

@mcp.tool(description=(
    "Create a new runbook in the Knowledge Base. Call this after diagnosing an incident. "
    "The content should follow the structure: ## Root Cause, ## Diagnostic Steps, "
    "## Resolution, ## Prevention. Markdown is supported."
))
def create_runbook(team_id: str, incident_id: str, title: str, content: str) -> dict:
    _logger.info("tool=create_runbook team_id=%s incident_id=%s", team_id, incident_id)
    return _client.create_runbook(team_id, incident_id, title, content)


@mcp.tool(description="Update the title and/or content of an existing active runbook.")
def update_runbook(runbook_id: str, title: Optional[str] = None, content: Optional[str] = None) -> dict:
    _logger.info("tool=update_runbook runbook_id=%s", runbook_id)
    return _client.update_runbook(runbook_id, title, content)


@mcp.tool(description=(
    "Mark a runbook as deprecated. It is retained for audit purposes "
    "but excluded from active runbook listings."
))
def deprecate_runbook(runbook_id: str) -> dict: 
    _logger.info("tool=deprecate_runbook runbook_id=%s", runbook_id)
    return _client.deprecate_runbook(runbook_id)


@mcp.tool(description="Retrieve a single runbook by ID, including its full content.")
def get_runbook(runbook_id: str) -> dict:
    _logger.info("tool=get_runbook runbook_id=%s", runbook_id)
    return _client.get_runbook(runbook_id)


@mcp.tool(description=(
    "List runbooks for a team. Filter by status: 'active' (default) or 'deprecated'. "
    "Returns id, title, incident_id, status, updated_at only — call get_runbook() for full content."
))
def list_runbooks(team_id: str, status: Optional[str] = "active") -> list: 
    _logger.info("tool=list_runbooks team_id=%s status=%s", team_id, status)
    return _client.list_runbooks(team_id, status)    


@mcp.tool(description=(
    "Retrieve a cleansed incident context optimised for runbook generation. "
    "Returns: incident summary, affected services with key metrics (noise-filtered), "
    "a timeline of up to 3 recent status transitions, and any existing runbooks for this incident. "
    "Use this before calling create_runbook()."
))
def get_incident(incident_id: str) -> dict: 
    _logger.info("tool=get_incident incident_id=%s", incident_id)
    return _client.get_incident_context(incident_id)


@mcp.tool(description=(
    "List all team incidents with summary info (id, title, status, age). "
    "Use get_incident() on a specific ID to get the full enriched context."
))
def list_incidents(team_id: str) -> list:
    _logger.info("tool=list_incidents team_id=%s", team_id)
    return _client.list_incidents(team_id)
