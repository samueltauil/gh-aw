---
mcp-servers:
  datadog:
    url: "https://mcp.datadoghq.com/api/unstable/mcp-server/mcp"
    headers:
      DD_API_KEY: "${{ secrets.DD_API_KEY }}"
      DD_APPLICATION_KEY: "${{ secrets.DD_APPLICATION_KEY }}"
      DD_SITE: "${{ secrets.DD_SITE || 'datadoghq.com' }}"
    allowed:
      - search_datadog_dashboards
      - search_datadog_slos
      - search_datadog_metrics
      - get_datadog_metric
      - analyze_datadog_logs
      - search_datadog_logs
      - search_datadog_events
      - search_datadog_monitors
      - search_datadog_incidents
      - get_datadog_incident
      - search_datadog_hosts
      - search_datadog_services
      - search_datadog_spans
      - get_datadog_trace
      - search_datadog_notebooks
      - search_datadog_rum_events
---

<!--

Datadog MCP Server
Observability and monitoring platform integration

Provides comprehensive access to Datadog monitoring, logs, metrics, and incidents
Documentation: https://docs.datadoghq.com/bits_ai/mcp_server/

This shared configuration provides the official Datadog MCP server integration for
monitoring, observability, and log analysis via HTTP API.

Available tools:
  Logs & Events:
  - analyze_datadog_logs: SQL-like pattern analysis on logs
  - search_datadog_logs: Raw log retrieval with query filtering and time ranges
  - search_datadog_events: Event querying within a time range
  - search_datadog_rum_events: Real User Monitoring (frontend) event search

  Metrics & Dashboards:
  - get_datadog_metric: Metric time series data retrieval
  - search_datadog_metrics: List and search available metrics
  - search_datadog_dashboards: Find and view dashboards

  APM (Tracing):
  - get_datadog_trace: Retrieve a trace by ID
  - search_datadog_spans: Span-level trace analytics

  Incidents & Monitoring:
  - search_datadog_incidents: List and query incidents
  - get_datadog_incident: Get details of a specific incident
  - search_datadog_monitors: Query and view monitor status

  Infrastructure & Services:
  - search_datadog_hosts: Host inventory and status
  - search_datadog_services: Service listing and dependency mapping

  Notebooks:
  - search_datadog_notebooks: Investigative notebook lookup

  SLOs:
  - search_datadog_slos: Search and view Service Level Objectives
#
Setup:
  1. Create Datadog API Keys:
     - Log in to your Datadog account
     - Go to Organization Settings > API Keys to create an API key
     - Go to Organization Settings > Application Keys to create an application key
#
  2. Add Repository Secrets:
     - DD_API_KEY: Your Datadog API key (required)
     - DD_APPLICATION_KEY: Your Datadog Application key (required)
     - DD_SITE: Your Datadog site domain (optional, defaults to datadoghq.com)
#
  3. Include in Your Workflow:
     imports:
       - shared/mcp/datadog.md
#
Regional Endpoints:
  The DD_SITE secret should match your Datadog region:
  - US (Default): datadoghq.com
  - EU: datadoghq.eu
  - US3 (GovCloud): ddog-gov.com
  - US5: us5.datadoghq.com
  - AP1: ap1.datadoghq.com
#
Example Usage:
  Search for error logs in the web-app service from the last hour and 
  summarize the most common errors.
#
Connection Type:
  This configuration uses HTTP MCP server type, connecting directly to the 
  official Datadog MCP API endpoint. Authentication is handled via HTTP headers.
  The server is in official preview; organizations must be allowlisted.
#
Troubleshooting:
  403 Forbidden Errors - Verify that:
  - Your API key and Application key are correct
  - The keys have necessary permissions to access requested resources
  - You're using the correct endpoint for your region
  - Your Datadog account has access to the requested data
  - Your organization is allowlisted for the Datadog MCP preview
#
Usage:
  imports:
    - shared/mcp/datadog.md

-->
