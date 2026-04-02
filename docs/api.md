# HELM API Documentation

## Overview

HELM provides a RESTful API for managing AI coding agent sessions, costs, memory, and more.

**Base URL:** `http://localhost:8080/api`

## Authentication

All API endpoints require authentication via Bearer token:

```
Authorization: Bearer <token>
```

## Endpoints

### Sessions

#### GET /api/sessions
List all sessions.

**Query Parameters:**
- `limit` (int): Number of sessions to return (default: 50)
- `offset` (int): Offset for pagination
- `status` (string): Filter by status (running, done, failed, paused)

**Response:**
```json
[
  {
    "id": "abc123",
    "provider": "anthropic",
    "model": "claude-sonnet-4-20250514",
    "project": "/path/to/project",
    "status": "done",
    "cost": 0.05,
    "started_at": "2024-01-01T00:00:00Z"
  }
]
```

#### GET /api/sessions/{id}
Get session details.

**Response:**
```json
{
  "id": "abc123",
  "provider": "anthropic",
  "model": "claude-sonnet-4-20250514",
  "project": "/path/to/project",
  "prompt": "Implement feature X",
  "status": "done",
  "cost": 0.05,
  "input_tokens": 1000,
  "output_tokens": 500,
  "started_at": "2024-01-01T00:00:00Z",
  "ended_at": "2024-01-01T00:05:00Z"
}
```

### Cost

#### GET /api/cost
Get cost information.

**Query Parameters:**
- `project` (string): Project path (default: current)
- `period` (string): Period (today, week, month)

**Response:**
```json
{
  "today": {"total_cost": 1.50, "input_tokens": 10000, "output_tokens": 5000},
  "week": {"total_cost": 10.00, "input_tokens": 50000, "output_tokens": 25000},
  "month": {"total_cost": 40.00, "input_tokens": 200000, "output_tokens": 100000},
  "budget": {"daily_limit": 10.0, "weekly_limit": 50.0, "monthly_limit": 150.0}
}
```

### Memory

#### GET /api/memory
List project memory entries.

**Query Parameters:**
- `project` (string): Project path
- `type` (string): Filter by type

**Response:**
```json
[
  {
    "id": "mem123",
    "project": "/path/to/project",
    "type": "convention",
    "key": "naming",
    "value": "Use PascalCase for exported types",
    "confidence": 0.85,
    "usage_count": 15
  }
]
```

### Analytics

#### GET /api/analytics
Get model performance analytics.

**Response:**
```json
[
  {
    "model": "claude-sonnet-4-20250514",
    "task_type": "code_generation",
    "attempts": 100,
    "successes": 95,
    "total_cost": 5.00,
    "avg_tokens": 2000,
    "avg_time_seconds": 30
  }
]
```

## Health Endpoints

### GET /health
Overall health check.

### GET /health/ready
Readiness probe.

### GET /health/live
Liveness probe.

### GET /metrics
Prometheus metrics endpoint.

## Error Responses

```json
{
  "error": "error message",
  "code": "ERROR_CODE"
}
```

**Error Codes:**
- `UNAUTHORIZED` (401): Invalid or missing authentication
- `FORBIDDEN` (403): Insufficient permissions
- `NOT_FOUND` (404): Resource not found
- `INTERNAL` (500): Internal server error
