4.3	System Requirement Specification
4.3.1	Functional Requirements Specification (FR)
4.3.1.1	User Authentication Module

Backend Authentication (Email/Password)
•	FR-01: The system shall store user passwords using a hashing algorithm (bcrypt) before saving them to the PostgreSQL database.
•	FR-02: The system shall validate login credentials and issue a signed JWT token upon successful authentication.
•	FR-03: The system shall provide a Logout function that invalidates the active user session and redirects the user to the login page.
•	FR-04: The system shall support Time-Based One-Time Password (TOTP) multi-factor authentication for registered users. The system shall allow users to set up, verify, and disable backend TOTP via dedicated endpoints.
•	FR-05: The system shall allow new users to register an account by providing a username, email, and password.
•	FR-06: The system shall allow registered users to log in with their email or username and password, followed by TOTP verification if MFA is enabled.

Firebase Authentication
•	FR-07: The system shall support Firebase Authentication as an alternative sign-in and registration provider. The login and registration pages shall offer a toggle between backend authentication and Firebase authentication modes.
•	FR-08: Upon a successful Firebase sign-in, the frontend shall exchange the Firebase ID token with the backend via a token exchange endpoint (/auth/exchange) to obtain a backend-issued JWT session. The backend shall verify the Firebase ID token using the Firebase Admin SDK before issuing the JWT.
•	FR-09: If no local user record exists for the authenticated Firebase UID, the system shall automatically provision a new user account seeded with the email and display name extracted from the Firebase ID token claims, with a default scope of "engineer".
•	FR-10: Firebase-authenticated users shall be required to verify their email address (via Firebase email verification) before they are permitted to proceed to TOTP setup.
•	FR-11: If backend TOTP is enabled for a Firebase-authenticated user, the token exchange endpoint shall require a valid TOTP code alongside the Firebase ID token before issuing the backend JWT.
•	FR-12: The frontend API client shall silently refresh the Firebase ID token and retry the token exchange when a 401 Unauthorized response is received, to maintain a seamless session without requiring the user to manually re-authenticate.

4.3.1.2	Real-Time Interaction Chat Interface Module
•	FR-13: The system shall provide a natural language chat interface for support engineers to query incident details, alerts, and runbooks.
•	FR-14: The system shall stream AI-generated responses and reasoning steps from the backend to the frontend token-by-token to reduce perceived latency.
•	FR-15: The system shall display reasoning steps of the agent to inform the user which tools or MCP servers are being accessed during a response.
•	FR-16: The system shall classify each user message as either a Task intent or a Conversational intent before deciding whether to expose LLM tools. Tool declarations shall only be sent to the LLM when the intent is classified as Task.
•	FR-17: The system shall provide a slash command interface within the chat composer that is triggered when the user types "/". The following slash commands shall be supported:
  - /alert              — List recent telemetry alerts ingested by the system with their severity and link status.
  - /incident [query]   — Show the active session incident, look up by surrogate key (INC-xxx) or UUID, or search team incidents by keyword.
  - /runbook [query]    — List active team runbooks, or search runbooks by keyword across title and content.
  - /quit               — Stop ongoing LLM processing immediately.
  - /help               — List all registered slash commands with their descriptions.
•	FR-18: The system shall display a command palette pop-up when the user types "/" in the chat input, showing matching slash commands with their usage hints. The palette shall be dismissed when the user selects a command or presses Esc.
•	FR-19: The system shall persist chat conversations and messages per team and user, and allow engineers to browse and restore past chat sessions from a side panel.
•	FR-20: The system shall automatically generate a short conversation title (5 words or fewer) from the first user prompt and assistant reply using the LLM, and store it with the conversation record.
•	FR-21: The system shall support dark mode and light mode theme switching within the chat interface.

4.3.1.3	Alert Ingestion Module
•	FR-22: The system shall expose a POST endpoint to receive JSON-formatted alert payloads from external monitoring platforms.
•	FR-23: The system shall parse incoming alert payloads to extract and store the following sections: Alert metadata (ID, severity, timestamps), Resource context (service, cluster, namespace), Telemetry metrics (CPU, memory, error rate, latency), Business context, and Metadata.
•	FR-24 [Optional]: The system shall automatically initiate an investigation workflow upon receipt of alerts with a warning or critical severity status.

4.3.1.4	Alert Validation and Classification Module (MCP-1)
•	FR-25: The system shall call the MCP-1 Server with an alert payload when the user explicitly requests alert validation and provides an alert ID.
•	FR-26: The system shall utilize the Isolation Forest machine learning model via MCP-1 to calculate an anomaly score and produce a classification label (Real Alert or False Alarm) for the alert.
•	FR-27: The system shall classify the alert result as either a 🚨 Real Alert (status: 0 — Confirmed Anomaly) or 🟢 False Alarm (status: 1 — System Healthy) based on the ML prediction returned by MCP-1.
•	FR-28: The system shall perform analysis on incoming alerts based on shared metadata such as the same Host ID or Service Name to enable correlated alert detection.
•	FR-29: The agent shall not call validate_alert automatically for mentions of an alert ID without an explicit user request for validation. It shall instead guide the user to request validation first.
•	FR-30: The agent shall guard against calling validate_alert with placeholder, empty, null, or fabricated alert IDs. If the alert ID is missing or invalid, the agent shall ask the user to provide a valid alert ID before proceeding.

4.3.1.5	Incident Diagnosis & Knowledge Retrieval Module (MCP-2)
•	FR-31: If an alert is validated as a Real Alert (status: 0), the LLM agent shall provide a technical diagnosis and recommended remediation steps based on the MCP-1 validation result. The agent may subsequently access the MCP-2 Server to retrieve or create runbooks when the engineer explicitly requests runbook-related actions (e.g. creating, listing, or fetching a runbook).
•	FR-32: The agent shall retrieve runbook records from MCP-2 (via get_runbook or get_incident) and analyze their procedures to formulate an attributed remediation plan.
•	FR-33: The agent shall attribute generated runbooks to the authenticated user who initiated the investigation.
•	FR-34: For a 🚨 Real Alert (status: 0), the LLM agent shall formulate actionable diagnosis and remediation recommendations based on the telemetry and ML anomaly payload, leaving incident linkage to engineer discretion via the incident management workspace.
•	FR-35: For a 🟢 False Alarm (status: 1), the LLM agent shall conclude that system telemetry is healthy, reassure the engineer that no incident linking or escalation is needed, and suppress remediation or troubleshooting steps.

4.3.1.6	Write-Back Mechanism Module
•	FR-36: The system shall allow the engineer to create a new runbook via the LLM agent (create_runbook tool) or through the Runbooks panel in the UI.
•	FR-37: The system shall save finalized runbooks using MCP-2 for future retrieval and reference.
•	FR-38: The system shall allow the user to archive (deprecate) an existing runbook via the LLM agent (deprecate_runbook tool) or through the UI to maintain knowledge base relevance.
•	FR-39: The system shall allow the user to manually update or refine an LLM-generated runbook via the LLM agent (update_runbook tool) or through the Runbooks panel before finalization.
•	FR-40: The system shall allow the user to view an individual runbook's full content and details through the Runbooks panel in the UI.

4.3.1.7	Team-Based RBAC Module
•	FR-41: Any registered engineer shall be able to create a team. The creating engineer is automatically assigned the "owner" role in that team.
•	FR-42: The team owner shall be able to invite other registered engineers to join the team by adding their email address. Only the team owner may add new members.
•	FR-43: All users within the same team shall have access to shared resources, including historical incident analysis, open and closed incident records, runbooks, and chat history scoped to that team.
•	FR-44: Engineers shall be able to switch their active team context via a dropdown selector in the header, which updates accessible incidents, chat history, and runbooks accordingly.
•	FR-45: Any member within a team shall be able to view, create, and edit a shared set of Support Copilot custom instructions that define remediation workflows and preferences for the agent within that workspace.
•	FR-46: The system shall validate that any new or edited instruction contains a minimum of 30 characters before saving.
•	FR-47: The system shall maintain an audit log of instruction changes, recording the previous instruction content and the user who made the change each time the instruction is saved.
•	FR-48: The system shall inject the active team's custom instruction into the agent's system prompt at runtime so the LLM adheres to team-defined workflows for every response within that workspace.

4.3.1.8	Incident Management Module
•	FR-49: The system shall allow the engineer to view a list of all open and closed incidents belonging to their active team via the Incidents panel.
•	FR-50: The system shall allow the engineer to view the full details of an existing incident, including title, status, severity, age, summary, telemetry, and linked runbooks.
•	FR-51: The system shall allow the engineer to manually create a new incident record with a title, description, service, and severity from the Incidents panel.
•	FR-52: The system shall allow the engineer to update an existing incident's title, status, or details.
•	FR-53: The system shall support incident surrogate keys in the format INC-xxx (e.g. INC-101) that are human-readable and usable directly in slash commands and LLM tool calls.
•	FR-54: The system shall allow the engineer to view all telemetry alerts currently linked to an incident within the incident workspace thread.
•	FR-55: The system shall allow the engineer to attach one or multiple unlinked telemetry alerts to an incident (supporting many-to-many relationships) via the Link Alerts modal.
•	FR-56: The system shall allow the engineer to unlink any associated alert from an incident with immediate UI feedback and database dissociation via the REST API (DELETE /api/incidents/:id/alerts/:alert_id).

4.3.1.9	Analytics Dashboard Module
•	FR-57: The system shall provide an analytics dashboard that displays incident trend data over selectable time periods (day, month, year) for the active team.
•	FR-58: The system shall compute and display the Mean Time To Resolve (MTTR) metric and SLA compliance rate for the active team based on closed incident records.
•	FR-59: The system shall display a list of SLA-breached incidents (incidents resolved beyond the SLA target time) with pagination support.
•	FR-60: The system shall provide a super_admin view of the analytics dashboard that aggregates incident trend, MTTR, and SLA breach data across all teams.

4.3.2	Non Functional Requirement Specification (NFR)
4.3.2.1	Security
•	NFR-01 (Password Policy): User passwords shall contain at least 6 characters and a maximum of 8 characters, including at least one special character.
•	NFR-02 (Data Privacy): The system shall redact or process PII-related data before querying the LLM.
•	NFR-03 (Authentication Guard): All protected API endpoints shall require a valid JWT token. The system shall validate the JWT and extract the user identity on every request via middleware before routing to the handler.
•	NFR-04 (Tool Call Guard): The agent shall reject tool calls that contain dummy, null, placeholder, or empty required parameters. If required parameters are missing, the agent shall fall back to a plain-text response prompting the user for the missing information instead of fabricating values.

4.3.2.2	Performance
•	NFR-05 (Response Time): The Chat Interface shall begin streaming the AI response within 10 seconds of the user submitting a query or an automated investigation trigger.
•	NFR-06 (Instant Feedback): Upon receiving a user message, the system shall emit a reasoning event immediately (e.g. "🧠 Analyzing prompt and evaluating available tools...") before initiating the LLM call, so the user has visible feedback with zero perceived idle time.

4.3.2.3	Reliability & Availability
•	NFR-07 (Graceful Degradation): If an MCP server (MCP-1 or MCP-2) is unreachable or returns an error, the system shall surface a clear error notification within the chat interface, allowing the engineer to continue manual investigation rather than freezing the application.
•	NFR-08 (Database Availability): The system shall maintain consistent database connections with a connection pool to ensure that high-frequency alert ingestion does not exhaust PostgreSQL resources.
•	NFR-09 (Connection Resilience): The system shall implement a backoff strategy for streaming reconnections to ensure the UI remains synchronized during brief network interruptions.
•	NFR-10 (Embedded Tool Call Recovery): If the LLM emits a tool call as raw JSON text content instead of using the standard tool_calls mechanism, the system shall detect and parse the embedded call before execution, with a graceful fallback to a plain-text re-query if parsing fails.

4.3.2.4	Explainability
•	NFR-11 (Source Attribution): Every recommendation or diagnosis by the agent must include a reference to its specific source — alert validation results must cite MCP-1 by name, and runbook or knowledge base content must cite MCP-2 and the specific document name. The agent shall not use vague phrases like "the system" or "the data" in place of the explicit source name.
•	NFR-12 (No Fabrication): The agent shall never fabricate, guess, or hallucinate alert IDs, incident IDs, runbook IDs, anomaly scores, metrics, or runbook steps when a tool returns an error, times out, or indicates that a record was not found.
•	NFR-13 (No Filler Text): The agent shall not generate holding messages such as "I am checking the database..." or "Please hold on..." before making a tool call. The agent shall call the tool silently and only respond after the result is received.
