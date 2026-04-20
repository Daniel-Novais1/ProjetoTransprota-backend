# API Documentation - TranspRota Backend

**Version:** 1.0.0  
**Base URL:** `http://localhost:8080`  
**Last Updated:** 2026-04-13

---

## Table of Contents

1. [Authentication](#authentication)
2. [Health Check](#health-check)
3. [Telemetry Endpoints](#telemetry-endpoints)
4. [Analytics Endpoints](#analytics-endpoints)
5. [WebSocket](#websocket)
6. [Error Responses](#error-responses)
7. [JSON Format Convention](#json-format-convention)

---

## Authentication

### POST /api/v1/auth/login

Authenticate user and receive JWT token.

**Request:**
```json
{
  "username": "admin",
  "password": "admin123"
}
```

**Response (200 OK):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "userId": "admin",
  "username": "admin"
}
```

**Error Responses:**
- `400 Bad Request`: Invalid request format
- `401 Unauthorized`: Invalid credentials
- `500 Internal Server Error`: Failed to generate token

---

## Health Check

### GET /health
### GET /api/v1/health

Check server health and database connectivity.

**Response (200 OK):**
```json
{
  "status": "healthy",
  "database": "connected",
  "redis": "connected",
  "timestamp": "2026-04-13T12:00:00Z"
}
```

---

## Telemetry Endpoints

### POST /api/v1/telemetry/gps

Receive GPS telemetry ping from device. Requires JWT authentication.

**Headers:**
- `Authorization: Bearer {token}`

**Request:**
```json
{
  "deviceId": "device-123",
  "recordedAt": "2026-04-13T12:00:00Z",
  "lat": -16.6869,
  "lng": -49.2648,
  "speed": 45.5,
  "heading": 180.0,
  "accuracy": 10.0,
  "transportMode": "bus",
  "routeId": "R-001",
  "batteryLevel": 85,
  "platform": "android",
  "appVersion": "4.0.0"
}
```

**Validation Rules:**
- `deviceId`: Required, string
- `recordedAt`: Required, RFC3339 format, not in future, not older than 24h
- `lat`: Required, float, valid range for Goiânia region
- `lng`: Required, float, valid range for Goiânia region
- `speed`: Optional, float, 0-120 km/h (urban limit)
- `heading`: Optional, float, 0-360 degrees
- `accuracy`: Optional, float, 1-100 meters
- `transportMode`: Optional, enum: `bus`, `car`, `bike`, `walk`, `metro`
- `routeId`: Optional, string
- `batteryLevel`: Optional, int, 0-100
- `platform`: Optional, enum: `android`, `ios`
- `appVersion`: Optional, string

**Response (202 Accepted):**
```json
{
  "status": "accepted",
  "telemetryId": "tel-1234567890",
  "deviceHash": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "cached": false,
  "processedAt": "2026-04-13T12:00:00Z",
  "ttlSeconds": 300
}
```

**Error Responses:**
- `400 Bad Request`: Invalid JSON payload
- `422 Unprocessable Entity`: Validation failed
- `401 Unauthorized`: Missing or invalid token

---

### GET /api/v1/telemetry/last-position/:device_hash

Get last known position of a specific device. Public endpoint (no auth required).

**Path Parameters:**
- `device_hash`: 32-character hash string

**Response (200 OK):**
```json
{
  "source": "redis",
  "position": {
    "lng": -49.2648,
    "lat": -16.6869,
    "speed": 45.5,
    "heading": 180.0,
    "accuracy": 10.0,
    "transportMode": "bus",
    "routeId": "R-001",
    "batteryLevel": 85,
    "recordedAt": 1715625600,
    "createdAt": 1715625660
  },
  "cached": true
}
```

**Error Responses:**
- `400 Bad Request`: Invalid device hash format
- `404 Not Found`: No position found for device

---

### GET /api/v1/telemetry/latest

Get last known position of all devices. Public endpoint (no auth required). Uses Redis-First strategy with PostgreSQL fallback.

**Response (200 OK):**
```json
{
  "count": 10,
  "positions": [
    {
      "deviceHash": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
      "routeId": "R-001",
      "speed": 45.5,
      "location": {
        "lat": -16.6869,
        "lng": -49.2648
      },
      "recordedAt": "2026-04-13T12:00:00Z",
      "platform": "android",
      "appVersion": "4.0.0",
      "trafficStatus": "fluido"
    }
  ],
  "source": "redis"
}
```

**Note:** Returns empty array `[]` with `count: 0` if no data available.

---

### GET /api/v1/telemetry/alerts

Get recent geofencing alerts (vehicles out of route or stopped outside terminal). Public endpoint.

**Query Parameters:**
- `limit`: Optional, int, 1-500 (default: 50)

**Response (200 OK):**
```json
{
  "count": 5,
  "alerts": [
    {
      "id": 1,
      "deviceHash": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
      "geofenceId": 1,
      "geofenceNome": "Terminal Central",
      "estado": "Out",
      "lat": -16.6869,
      "lng": -49.2648,
      "ocorridoEm": "2026-04-13T11:30:00Z"
    }
  ]
}
```

---

### GET /api/v1/telemetry/eta/:device_hash

Calculate Estimated Time of Arrival (ETA) to a destination. Public endpoint.

**Path Parameters:**
- `device_hash`: 32-character hash string

**Query Parameters:**
- `lat`: Required, float, destination latitude
- `lng`: Required, float, destination longitude

**Response (200 OK):**
```json
{
  "device_hash": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "destination": {
    "lat": -16.6900,
    "lng": -49.2700
  },
  "current_position": {
    "lat": -16.6869,
    "lng": -49.2648
  },
  "distance_meters": 650.5,
  "distance_km": 0.65,
  "avg_speed_kmh": 40.0,
  "eta_minutes": 0.98,
  "eta_seconds": 58.5,
  "calculated_at": "2026-04-13T12:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request`: Missing destination coordinates or invalid format
- `500 Internal Server Error`: Failed to calculate distance or retrieve position

---

### GET /api/v1/telemetry/history

Get historical trajectory of a device within a time range. Requires JWT authentication.

**Headers:**
- `Authorization: Bearer {token}`

**Query Parameters:**
- `device_id`: Required, string
- `start`: Optional, RFC3339 format (default: 1 hour ago)
- `end`: Optional, RFC3339 format (default: now)

**Constraints:**
- Maximum time range: 7 days

**Response (200 OK):**
```json
{
  "device_id": "device-123",
  "start": "2026-04-13T11:00:00Z",
  "end": "2026-04-13T12:00:00Z",
  "count": 250,
  "points": [
    {
      "lat": -16.6869,
      "lng": -49.2648,
      "speed": 45.5,
      "recordedAt": "2026-04-13T11:00:00Z"
    }
  ],
  "decimated": false,
  "retrieved_at": "2026-04-13T12:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request`: Missing device_id, invalid time format, or time range exceeds 7 days
- `401 Unauthorized`: Missing or invalid token
- `500 Internal Server Error`: Failed to retrieve history

---

### GET /api/v1/telemetry/export

Export historical trajectory in CSV format. Requires JWT authentication.

**Headers:**
- `Authorization: Bearer {token}`

**Query Parameters:**
- `device_id`: Required, string
- `start`: Optional, RFC3339 format (default: 24 hours ago)
- `end`: Optional, RFC3339 format (default: now)

**Constraints:**
- Maximum time range: 7 days

**Response (200 OK):**
- Content-Type: `text/csv`
- CSV format: `lat,lng,speed,recorded_at`

**Error Responses:**
- `400 Bad Request`: Missing device_id, invalid time format, or time range exceeds 7 days
- `401 Unauthorized`: Missing or invalid token
- `500 Internal Server Error`: Failed to retrieve history

---

### GET /api/v1/telemetry/fleet-status

Get fleet status metrics for CCO (Central Control Operations). Requires JWT authentication. Cached for 2 minutes.

**Headers:**
- `Authorization: Bearer {token}`

**Response (200 OK):**
```json
{
  "totalActiveBuses": 45,
  "totalGeofenceAlerts": 3,
  "averageSpeed": 38.5,
  "totalBuses": 50,
  "offlineBuses": 5,
  "lastUpdated": "2026-04-13T12:00:00Z"
}
```

**Error Responses:**
- `401 Unauthorized`: Missing or invalid token
- `500 Internal Server Error`: Failed to retrieve fleet status

---

## Analytics Endpoints

### GET /api/v1/analytics/fleet-health

Get fleet health metrics (efficiency, dwell time, etc.). Cached for 15 minutes.

**Query Parameters:**
- `hours`: Optional, int, 1-168 (default: 24, max: 7 days)

**Response (200 OK):**
```json
{
  "hours": 24,
  "count": 50,
  "source": "cache",
  "metrics": [
    {
      "deviceHash": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
      "routeId": "R-001",
      "movingTimeMin": 420.5,
      "totalTimeMin": 480.0,
      "efficiency": 87.6,
      "dwellTimeMin": 59.5,
      "avgSpeedKmh": 38.2,
      "totalPings": 1250
    }
  ],
  "calculated_at": "2026-04-13T12:00:00Z"
}
```

**Error Responses:**
- `500 Internal Server Error`: Failed to retrieve fleet health metrics

---

## WebSocket

### GET /api/v1/telemetry/ws

WebSocket endpoint for real-time GPS updates.

**Connection URL:**
```
ws://localhost:8080/api/v1/telemetry/ws
```

**Message Format (Server → Client):**
```json
{
  "id": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "lat": -16.6869,
  "lng": -49.2648,
  "speed": 45.5,
  "traffic_status": "fluido",
  "timestamp": 1715625600
}
```

**Note:** Messages are published via Redis Pub/Sub whenever new GPS telemetry is received.

---

## Error Responses

### Standard Error Format

All errors follow this format:

```json
{
  "error": "error_type_or_message"
}
```

### Common Error Codes

- `400 Bad Request`: Invalid request parameters or format
- `401 Unauthorized`: Missing or invalid authentication token
- `404 Not Found`: Resource not found
- `422 Unprocessable Entity`: Validation failed
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Internal server error

### JWT Authentication Errors

- `{"error": "unauthorized"}`: Invalid or missing token
- `{"error": "expired"}`: Token has expired

---

## JSON Format Convention

All API responses use **camelCase** for JSON field names to match JavaScript/React conventions.

**Examples:**
- `device_id` → `deviceId`
- `recorded_at` → `recordedAt`
- `transport_mode` → `transportMode`
- `route_id` → `routeId`
- `battery_level` → `batteryLevel`
- `app_version` → `appVersion`
- `created_at` → `createdAt`
- `ocorrido_em` → `ocorridoEm`

This eliminates the need for manual data transformations in the frontend React application.

---

## Rate Limiting

- Default: 100 requests per minute per IP
- Device-specific: 60 requests per minute per device
- Exceeded limit returns `429 Too Many Requests`

---

## Caching Strategy

### Redis Cache

- **Last Position:** 5 minutes TTL (`last_pos:{device_hash}`)
- **Fleet Health:** 15 minutes TTL (`analytics:fleet_health:{hours}`)
- **Fleet Status:** 2 minutes TTL (`compliance:fleet_status`)
- **Compliance Queries:** 2 minutes TTL (`compliance:*`)

### Cache Invalidation

Compliance cache is automatically invalidated whenever a new audit log entry is created or updated.

---

## Security Features

### LGPD Compliance

- Device IDs are anonymized using SHA-256 hash with daily salt rotation
- Only device hashes are stored, not original device IDs
- Audit logs track all security events

### Validation

- GPS coordinates validated for Goiânia region bounds
- Speed limits enforced (max 120 km/h urban)
- Timestamp validation (not in future, not older than 24h)
- Transport mode validation (bus, car, bike, walk, metro)
- Battery level validation (0-100)

### Anomaly Detection

- Speed > 150 km/h flagged as anomaly
- Teleport detection (distance > 10km in 10s)
- All anomalies logged to audit trail

---

## Support

For issues or questions, contact the development team.
