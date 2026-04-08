import React from "react";
import { MapContainer, TileLayer, Marker, Popup, Polyline } from "react-leaflet";
import { Icon } from "leaflet";
import "leaflet/dist/leaflet.css";

// Fix for default markers in react-leaflet
delete (Icon.Default.prototype as any)._getIconUrl;
Icon.Default.mergeOptions({
  iconRetinaUrl: "https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.7.1/images/marker-icon-2x.png",
  iconUrl: "https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.7.1/images/marker-icon.png",
  shadowUrl: "https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.7.1/images/marker-shadow.png",
});

interface BusMapProps {
  route?: any; // RouteResponse
  center?: [number, number];
}

export const BusMap: React.FC<BusMapProps> = ({ route, center = [-16.686, -49.264] }) => {
  // Coordenadas da Vila Pedroso como centro padrão
  const vilaPedroso: [number, number] = [-16.686, -49.264];
  const terminalCentro: [number, number] = [-16.686, -49.264]; // Aproximado
  const ufg: [number, number] = [-16.0000, -48.9500];

  // Rota da Linha 104: Vila Pedroso -> Terminal Centro -> UFG
  const routePositions: [number, number][] = [
    vilaPedroso,
    terminalCentro,
    ufg,
  ];

  return (
    <div className="w-full h-96 rounded-lg overflow-hidden border">
      <MapContainer
        center={center}
        zoom={13}
        style={{ height: "100%", width: "100%" }}
      >
        <TileLayer
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
        />

        {/* Marker para Vila Pedroso */}
        <Marker position={vilaPedroso}>
          <Popup>Vila Pedroso - Origem</Popup>
        </Marker>

        {/* Marker para Terminal Centro */}
        <Marker position={terminalCentro}>
          <Popup>Terminal Centro - Ponto de Integração</Popup>
        </Marker>

        {/* Marker para UFG */}
        <Marker position={ufg}>
          <Popup>UFG Campus Samambaia - Destino</Popup>
        </Marker>

        {/* Desenhar rota se existir */}
        {route && route.numero_linha === "104" && (
          <Polyline
            positions={routePositions}
            color="blue"
            weight={4}
            opacity={0.7}
          />
        )}
      </MapContainer>
    </div>
  );
};

    return () => clearInterval(interval);
  }, [autoRefresh, busId]);

  const openGoogleMaps = () => {
    if (!location) return;
    const url = `https://www.google.com/maps?q=${location.latitude},${location.longitude}`;
    window.open(url, "_blank");
  };

  return (
    <div className="w-full max-w-2xl mx-auto p-6 bg-white rounded-lg shadow-md">
      <h2 className="text-2xl font-bold mb-4 flex items-center gap-2">
        <MapPin className="w-6 h-6 text-blue-600" />
        Rastreador de Ônibus
      </h2>

      <div className="flex gap-2 mb-4">
        <input
          type="text"
          placeholder="ID do Ônibus (ex: BUS_001)"
          value={busId}
          onChange={(e) => setBusId(e.target.value)}
          onKeyPress={(e) => e.key === "Enter" && fetchBusData()}
          className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <button
          onClick={fetchBusData}
          disabled={loading}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 transition flex items-center gap-2"
        >
          <RotateCw className={`w-4 h-4 ${loading ? "animate-spin" : ""}`} />
          Buscar
        </button>
      </div>

      <div className="mb-4 flex items-center gap-2">
        <input
          type="checkbox"
          id="autoRefresh"
          checked={autoRefresh}
          onChange={(e) => setAutoRefresh(e.target.checked)}
          className="w-4 h-4"
        />
        <label htmlFor="autoRefresh" className="text-sm text-gray-700">
          🔄 Auto-atualizar a cada 5 segundos
        </label>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg flex gap-2 items-start">
          <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" />
          <span className="text-red-700">{error}</span>
        </div>
      )}

      {location && status && (
        <div className="space-y-4">
          <div>
            <StatusBadge
              status={status.status as "Em trânsito" | "No Terminal"}
              busId={busId}
              terminal={status.terminal}
            />
          </div>

          <div className="p-4 bg-gray-50 rounded-lg">
            <p className="text-sm text-gray-600 mb-2">📍 Localização Atual:</p>
            <p className="text-lg font-mono text-gray-800">
              {location.latitude.toFixed(6)}, {location.longitude.toFixed(6)}
            </p>
            <p className="text-xs text-gray-500 mt-2">
              Atualizado em: {new Date(location.timestamp).toLocaleTimeString("pt-BR")}
            </p>
          </div>

          {status.terminal && (
            <div className="p-4 bg-blue-50 rounded-lg border border-blue-200">
              <p className="text-sm font-semibold text-blue-900">
                Terminal Próximo: {status.terminal}
              </p>
              <p className="text-xs text-blue-700 mt-1">
                {status.distancia_metros?.toFixed(0)}m de distância
              </p>
            </div>
          )}

          <button
            onClick={openGoogleMaps}
            className="w-full px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition"
          >
            🗺️ Abrir no Google Maps
          </button>
        </div>
      )}

      {!location && !error && (
        <div className="text-center py-8 text-gray-500">
          Digite um ID de ônibus e clique em Buscar
        </div>
      )}
    </div>
  );
};

export default BusMap;
