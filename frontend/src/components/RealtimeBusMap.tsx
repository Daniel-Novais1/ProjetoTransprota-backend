import React, { useEffect } from 'react';
import { MapContainer, TileLayer, Marker, Popup, useMap } from 'react-leaflet';
import { Icon, LatLngBounds } from 'leaflet';
import { useRealtime, RealtimeData } from '@/hooks/useRealtime';
import 'leaflet/dist/leaflet.css';

// Fix para ícones do Leaflet no React
const busIcon = new Icon({
  iconUrl: 'https://cdn-icons-png.flaticon.com/512/3448/3448636.png',
  iconSize: [32, 32],
  iconAnchor: [16, 32],
  popupAnchor: [0, -32],
});

interface RealtimeBusMapProps {
  center?: [number, number];
  zoom?: number;
}

function MapUpdater({ data }: { data: RealtimeData }) {
  const map = useMap();

  useEffect(() => {
    if (data.buses.size > 0) {
      const bounds = new LatLngBounds(
        Array.from(data.buses.values()).map((bus) => [bus.lat, bus.lng] as [number, number])
      );
      map.fitBounds(bounds, { padding: [50, 50] });
    }
  }, [data.buses, map]);

  return null;
}

export default function RealtimeBusMap({ center = [-16.6869, -49.2648], zoom = 13 }: RealtimeBusMapProps) {
  const { data, isConnected, clearBIAlert, clearGeofenceAlert } = useRealtime();

  return (
    <div className="relative w-full h-full">
      <div className="absolute top-4 left-4 z-[1000] bg-white rounded-lg shadow-lg px-4 py-2">
        <div className="flex items-center gap-2">
          <div className={`w-3 h-3 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`} />
          <span className="text-sm font-medium">
            {isConnected ? 'Conectado' : 'Desconectado'}
          </span>
        </div>
        <div className="text-xs text-gray-500 mt-1">
          {data.buses.size} ônibus ativos
        </div>
      </div>

      <MapContainer
        center={center}
        zoom={zoom}
        style={{ height: '100%', width: '100%' }}
        className="rounded-lg"
      >
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />
        
        <MapUpdater data={data} />

        {Array.from(data.buses.values()).map((bus) => (
          <Marker
            key={bus.device_id}
            position={[bus.lat, bus.lng]}
            icon={busIcon}
          >
            <Popup>
              <div className="p-2">
                <h3 className="font-bold text-sm mb-2">{bus.device_id}</h3>
                <div className="text-xs space-y-1">
                  <div>📍 Lat: {bus.lat.toFixed(6)}</div>
                  <div>📍 Lng: {bus.lng.toFixed(6)}</div>
                  <div>🚀 Velocidade: {bus.speed.toFixed(1)} km/h</div>
                  <div>⏰ Atualizado: {new Date(bus.timestamp).toLocaleTimeString()}</div>
                </div>
              </div>
            </Popup>
          </Marker>
        ))}
      </MapContainer>
    </div>
  );
}
