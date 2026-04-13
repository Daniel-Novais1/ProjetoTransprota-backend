import React, { useState } from 'react';
import { MapContainer, TileLayer, Marker, Popup } from 'react-leaflet';
import { Icon } from 'leaflet';
import HistoryLayer from './HistoryLayer';
import HistorySelector from './HistorySelector';
import 'leaflet/dist/leaflet.css';

interface HistoryPoint {
  lat: number;
  lng: number;
  speed: number;
  recorded_at: string;
}

export default function HistoryDashboard() {
  const [historyPoints, setHistoryPoints] = useState<HistoryPoint[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSearch = async (deviceId: string, startTime: string, endTime: string) => {
    setLoading(true);
    setError(null);

    try {
      const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080';
      const response = await fetch(
        `${apiUrl}/api/v1/telemetry/history?device_id=${deviceId}&start=${startTime}&end=${endTime}`
      );

      if (!response.ok) {
        throw new Error('Failed to fetch history');
      }

      const data = await response.json();
      setHistoryPoints(data.points || []);
    } catch (err) {
      setError('Erro ao buscar histórico. Verifique o ID do dispositivo.');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const busIcon = new Icon({
    iconUrl: 'https://cdn-icons-png.flaticon.com/512/3448/3448636.png',
    iconSize: [32, 32],
    iconAnchor: [16, 32],
    popupAnchor: [0, -32],
  });

  const center = historyPoints.length > 0 
    ? [historyPoints[0].lat, historyPoints[0].lng] as [number, number]
    : [-16.6869, -49.2648];

  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
      {/* Painel de Controle */}
      <div className="lg:col-span-1">
        <HistorySelector onSearch={handleSearch} loading={loading} />

        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg mt-4">
            {error}
          </div>
        )}

        {historyPoints.length > 0 && (
          <div className="bg-white rounded-lg shadow-lg p-6 mt-4">
            <h3 className="text-lg font-bold text-gray-800 mb-2">
              Resumo do Histórico
            </h3>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-gray-600">Pontos:</span>
                <span className="font-medium">{historyPoints.length}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-600">Início:</span>
                <span className="font-medium">
                  {new Date(historyPoints[0].recorded_at).toLocaleString()}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-600">Fim:</span>
                <span className="font-medium">
                  {new Date(historyPoints[historyPoints.length - 1].recorded_at).toLocaleString()}
                </span>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Mapa */}
      <div className="lg:col-span-2">
        <div className="bg-white rounded-lg shadow-lg p-4 h-[600px]">
          <MapContainer
            center={center}
            zoom={13}
            style={{ height: '100%', width: '100%' }}
          >
            <TileLayer
              attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
              url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
            />

            {historyPoints.length > 0 && (
              <HistoryLayer points={historyPoints} />
            )}

            {historyPoints.length > 0 && (
              <>
                {/* Marcador de início */}
                <Marker position={[historyPoints[0].lat, historyPoints[0].lng]} icon={busIcon}>
                  <Popup>
                    <div className="p-2">
                      <h3 className="font-bold text-sm mb-2">Início</h3>
                      <div className="text-xs space-y-1">
                        <div>Lat: {historyPoints[0].lat.toFixed(6)}</div>
                        <div>Lng: {historyPoints[0].lng.toFixed(6)}</div>
                        <div>Velocidade: {historyPoints[0].speed.toFixed(1)} km/h</div>
                        <div>{new Date(historyPoints[0].recorded_at).toLocaleString()}</div>
                      </div>
                    </div>
                  </Popup>
                </Marker>

                {/* Marcador de fim */}
                <Marker 
                  position={[
                    historyPoints[historyPoints.length - 1].lat,
                    historyPoints[historyPoints.length - 1].lng
                  ]} 
                  icon={busIcon}
                >
                  <Popup>
                    <div className="p-2">
                      <h3 className="font-bold text-sm mb-2">Fim</h3>
                      <div className="text-xs space-y-1">
                        <div>Lat: {historyPoints[historyPoints.length - 1].lat.toFixed(6)}</div>
                        <div>Lng: {historyPoints[historyPoints.length - 1].lng.toFixed(6)}</div>
                        <div>Velocidade: {historyPoints[historyPoints.length - 1].speed.toFixed(1)} km/h</div>
                        <div>{new Date(historyPoints[historyPoints.length - 1].recorded_at).toLocaleString()}</div>
                      </div>
                    </div>
                  </Popup>
                </Marker>
              </>
            )}
          </MapContainer>
        </div>
      </div>
    </div>
  );
}
