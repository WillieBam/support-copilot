"""Multi-server FastMCP host entrypoint.

Mounts sub-servers (Server 1: Telemetry & Anomaly Detection, Server 2: Knowledge Base)
onto a single FastMCP instance exposing streamable HTTP transport on /mcp.
"""

import logging
import os
from fastmcp import FastMCP
from server_1.server import mcp as mcp1
from server_2.server import mcp as mcp2

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)

MCP_HOST = os.getenv("MCP_HOST", "0.0.0.0")
MCP_PORT = int(os.getenv("MCP_PORT", "9000"))
MCP_PATH = os.getenv("MCP_PATH", "/mcp")

host = FastMCP("support-copilot-mcp-host")
host.mount(mcp1)
host.mount(mcp2)

if __name__ == "__main__":
    host.run(transport="streamable-http", host=MCP_HOST, port=MCP_PORT, path=MCP_PATH)