// API Types based on API_DOC.md

export interface TelemetryPing {
  deviceId: string;
  recordedAt: string;
  lat: number;
  lng: number;
  speed?: number;
  heading?: number;
  accuracy?: number;
  transportMode?: 'bus' | 'car' | 'bike' | 'walk' | 'metro';
  routeId?: string;
  batteryLevel?: number;
  platform?: 'android' | 'ios';
  appVersion?: string;
}

export interface TelemetryResponse {
  status: 'accepted';
  telemetryId: string;
  deviceHash: string;
  cached: boolean;
  processedAt: string;
  ttlSeconds: number;
}

export interface DevicePosition {
  lng: number;
  lat: number;
  speed: number;
  heading: number;
  accuracy: number;
  transportMode: string;
  routeId: string;
  batteryLevel: number;
  recordedAt: number;
  createdAt: number;
}

export interface LastPositionResponse {
  source: 'redis' | 'postgres';
  position: DevicePosition;
  cached: boolean;
}

export interface LatestPositionsResponse {
  count: number;
  positions: LatestPosition[];
  source: string;
}

export interface LatestPosition {
  deviceHash: string;
  routeId: string;
  speed: number;
  location: {
    lat: number;
    lng: number;
  };
  recordedAt: string;
  platform: string;
  appVersion: string;
  trafficStatus: 'fluido' | 'moderado' | 'congestionado';
  confidence?: number;
}

export interface ETAResponse {
  device_hash: string;
  destination: {
    lat: number;
    lng: number;
  };
  current_position: {
    lat: number;
    lng: number;
  };
  distance_meters: number;
  distance_km: number;
  avg_speed_kmh: number;
  eta_minutes: number;
  eta_seconds: number;
  calculated_at: string;
}

export interface GeofenceAlert {
  id: number;
  deviceHash: string;
  geofenceId: number;
  geofenceNome: string;
  estado: 'In' | 'Out';
  lat: number;
  lng: number;
  ocorridoEm: string;
}

export interface AlertsResponse {
  count: number;
  alerts: GeofenceAlert[];
}

export interface FleetStatusResponse {
  totalActiveBuses: number;
  totalGeofenceAlerts: number;
  averageSpeed: number;
  totalBuses: number;
  offlineBuses: number;
  lastUpdated: string;
}

export interface HealthResponse {
  status: string;
  database: string;
  redis: string;
  timestamp: string;
}

// UI Types for the new components

export interface Route {
  id: string;
  number: string;
  name: string;
  direction: string;
  status: 'active' | 'inactive';
  eta?: number;
  trafficStatus?: 'fluido' | 'moderado' | 'congestionado';
}

export interface Stop {
  id: string;
  name: string;
  status: 'past' | 'current' | 'future' | 'end';
  eta?: number;
  estimatedArrival?: string;
}

export interface BusLocation {
  id: string;
  lat: number;
  lng: number;
  speed: number;
  heading: number;
  isRecent: boolean;
}

export interface UserProfile {
  name: string;
  avatar: string;
  level: string;
  xp: number;
  xpToNextLevel: number;
  streak: number;
}
