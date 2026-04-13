import React from 'react';
import { Polyline } from 'react-leaflet';
import { LatLngExpression } from 'leaflet';

interface HistoryPoint {
  lat: number;
  lng: number;
  speed: number;
  recorded_at: string;
}

interface HistoryLayerProps {
  points: HistoryPoint[];
  color?: string;
  weight?: number;
  opacity?: number;
}

// Decimation: reduz número de pontos para não travar o navegador
// Pula pontos muito próximos (distância < 10 metros)
function decimatePoints(points: HistoryPoint[], minDistanceMeters: number = 10): HistoryPoint[] {
  if (points.length <= 1000) {
    return points; // Não decimar se poucos pontos
  }

  const decimated: HistoryPoint[] = [points[0]];
  let lastPoint = points[0];

  for (let i = 1; i < points.length; i++) {
    const current = points[i];
    const distance = haversineDistance(
      lastPoint.lat,
      lastPoint.lng,
      current.lat,
      current.lng
    );

    if (distance >= minDistanceMeters) {
      decimated.push(current);
      lastPoint = current;
    }
  }

  // Sempre incluir o último ponto
  if (decimated[decimated.length - 1] !== points[points.length - 1]) {
    decimated.push(points[points.length - 1]);
  }

  return decimated;
}

// Fórmula de Haversine para calcular distância em metros
function haversineDistance(lat1: number, lng1: number, lat2: number, lng2: number): number {
  const R = 6371000; // Raio da Terra em metros
  const dLat = ((lat2 - lat1) * Math.PI) / 180;
  const dLng = ((lng2 - lng1) * Math.PI) / 180;

  const a =
    Math.sin(dLat / 2) * Math.sin(dLat / 2) +
    Math.cos((lat1 * Math.PI) / 180) *
      Math.cos((lat2 * Math.PI) / 180) *
      Math.sin(dLng / 2) *
      Math.sin(dLng / 2);

  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
  return R * c;
}

export default function HistoryLayer({
  points,
  color = '#3b82f6',
  weight = 4,
  opacity = 0.8,
}: HistoryLayerProps) {
  // Aplicar decimation se muitos pontos
  const displayPoints = decimatePoints(points);

  // Converter para formato LatLngExpression
  const positions: LatLngExpression[] = displayPoints.map(
    (point) => [point.lat, point.lng] as LatLngExpression
  );

  return (
    <Polyline
      positions={positions}
      color={color}
      weight={weight}
      opacity={opacity}
      lineCap="round"
      lineJoin="round"
    />
  );
}
