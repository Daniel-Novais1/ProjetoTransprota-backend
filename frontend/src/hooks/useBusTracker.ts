import { useState, useRef, useCallback, useEffect } from 'react';

interface BusTrackerConfig {
	enabled?: boolean;
	intervalMs?: number;
	apiUrl?: string;
	onError?: (error: Error) => void;
	onSuccess?: (data: any) => void;
}

interface BusPosition {
	lat: number;
	lng: number;
	speed: number;
	heading: number;
	timestamp: number;
}

const R = 6371000;
const toRad = (deg: number) => (deg * Math.PI) / 180;
const STOPPED_THRESHOLD_MS = 120000;
const STOPPED_INTERVAL_MS = 30000;
const MOVING_INTERVAL_MS = 5000;

const haversine = (lat1: number, lng1: number, lat2: number, lng2: number): number => {
	const φ1 = toRad(lat1);
	const φ2 = toRad(lat2);
	const Δφ = toRad(lat2 - lat1);
	const Δλ = toRad(lng2 - lng1);
	const a = Math.sin(Δφ / 2) * Math.sin(Δφ / 2) + Math.cos(φ1) * Math.cos(φ2) * Math.sin(Δλ / 2) * Math.sin(Δλ / 2);
	const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
	return R * c;
};

export function useBusTracker(
	userId: string,
	deviceHash: string,
	routeId: string,
	config: BusTrackerConfig = {}
) {
	const { enabled = true, intervalMs = 5000, apiUrl = '/api/v1/bus-update', onError, onSuccess } = config;
	const [isTracking, setIsTracking] = useState(false);
	const [lastPosition, setLastPosition] = useState<BusPosition | null>(null);
	const [isStopped, setIsStopped] = useState(false);
	const watchIdRef = useRef<number | null>(null);
	const lastPositionRef = useRef<BusPosition | null>(null);
	const lastSendRef = useRef<number>(0);
	const lastMovementRef = useRef<number>(Date.now());
	const currentIntervalRef = useRef<number>(intervalMs);

	const updateInterval = useCallback(() => {
		const now = Date.now();
		const timeSinceMovement = now - lastMovementRef.current;
		const newInterval = timeSinceMovement > STOPPED_THRESHOLD_MS ? STOPPED_INTERVAL_MS : MOVING_INTERVAL_MS;
		
		if (newInterval !== currentIntervalRef.current) {
			currentIntervalRef.current = newInterval;
			setIsStopped(timeSinceMovement > STOPPED_THRESHOLD_MS);
		}
	}, []);

	const sendPosition = useCallback(async (position: GeolocationPosition) => {
		const { latitude, longitude, speed, heading } = position.coords;
		const now = Date.now();
		
		if (now - lastSendRef.current < currentIntervalRef.current) return;
		
		if (lastPositionRef.current) {
			const distance = haversine(lastPositionRef.current.lat, lastPositionRef.current.lng, latitude, longitude);
			const timeDiff = (now - lastPositionRef.current.timestamp) / 1000;
			const calculatedSpeed = (distance / timeDiff) * 3.6;
			
			if (calculatedSpeed > 1) {
				lastMovementRef.current = now;
				updateInterval();
			}
		}
		
		const payload = { user_id: userId, device_hash: deviceHash, lat: latitude, lng: longitude, route_id: routeId, speed: speed || 0, heading: heading || 0, is_on_bus: true, occupancy: 'unknown', terminal_id: 't01' };
		
		try {
			const response = await fetch(apiUrl, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) });
			if (!response.ok) throw new Error(`HTTP ${response.status}`);
			const data = await response.json();
			onSuccess?.(data);
			setLastPosition({ lat: latitude, lng: longitude, speed: speed || 0, heading: heading || 0, timestamp: now });
			lastPositionRef.current = { lat: latitude, lng: longitude, speed: speed || 0, heading: heading || 0, timestamp: now };
			lastSendRef.current = now;
		} catch (error) {
			onError?.(error as Error);
		}
	}, [userId, deviceHash, routeId, apiUrl, onError, onSuccess, updateInterval]);

	const startTracking = useCallback(() => {
		if (!enabled || watchIdRef.current !== null) return;
		setIsTracking(true);
		watchIdRef.current = navigator.geolocation.watchPosition(
			(position) => sendPosition(position),
			(error) => onError?.(error),
			{ enableHighAccuracy: true, timeout: 10000, maximumAge: 5000 }
		);
	}, [enabled, sendPosition, onError]);

	const stopTracking = useCallback(() => {
		if (watchIdRef.current !== null) {
			navigator.geolocation.clearWatch(watchIdRef.current);
			watchIdRef.current = null;
			setIsTracking(false);
		}
	}, []);

	useEffect(() => {
		const interval = setInterval(updateInterval, 10000);
		return () => clearInterval(interval);
	}, [updateInterval]);

	return { isTracking, lastPosition, isStopped, startTracking, stopTracking, sendPosition };
}

export const fetchNearbyBuses = async (lat: number, lng: number, radius: number, route?: string): Promise<any> => {
	// Se route for um número parcial (ex: "001"), usar como filtro parcial
	// O backend fará o Contains para encontrar rotas como "EIXO-001"
	const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';
	const url = route ? `${API_BASE}/api/v1/bus-locations?lat=${lat}&lng=${lng}&radius=${radius}&route=${route}` : `${API_BASE}/api/v1/bus-locations?lat=${lat}&lng=${lng}&radius=${radius}`;
	console.log('🔍 [API] fetchNearbyBuses URL:', url);
	const response = await fetch(url);
	console.log('🔍 [API] fetchNearbyBuses Status:', response.status, response.statusText);
	const data = await response.json();
	console.log('🔍 [API] fetchNearbyBuses Response:', data);
	if (!response.ok) throw new Error(`HTTP ${response.status}`);
	return data;
};

export const fetchAllBuses = async (): Promise<any> => {
	const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';
	const url = `${API_BASE}/api/v1/buses/all`;
	console.log('🔍 [API] fetchAllBuses URL:', url);
	const response = await fetch(url);
	console.log('🔍 [API] fetchAllBuses Status:', response.status, response.statusText);
	const data = await response.json();
	console.log('🔍 [API] fetchAllBuses Response:', data);
	if (!response.ok) throw new Error(`HTTP ${response.status}`);
	return data;
};

export const fetchDensityReport = async (hours: number = 24): Promise<any> => {
	const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';
	const url = `${API_BASE}/api/v1/analytics/density?hours=${hours}`;
	console.log('🔍 [API] fetchDensityReport URL:', url);
	const response = await fetch(url);
	console.log('🔍 [API] fetchDensityReport Status:', response.status, response.statusText);
	const data = await response.json();
	console.log('🔍 [API] fetchDensityReport Response:', data);
	if (!response.ok) throw new Error(`HTTP ${response.status}`);
	return data;
};
