import React, { useState, useEffect, useRef } from 'react';
import { MapContainer, TileLayer, Polyline, Marker, Popup } from 'react-leaflet';
import { LatLngExpression, LatLngBounds, Icon, DivIcon } from 'leaflet';
import axios from 'axios';

interface MapPoint {
  name: string;
  lat: number;
  lng: number;
}

interface MapStep {
  name: string;
  lat: number;
  lng: number;
  is_terminal: boolean;
  is_transfer: boolean;
}

interface MapRouteResponse {
  origin: MapPoint;
  destination: MapPoint;
  steps: MapStep[];
  total_time_minutes: number;
  bus_lines: string[];
}

interface RoutePreset {
  id: number;
  name: string;
  origin: string;
  destination: string;
  description: string;
  complexity: string;
  estimated_time: string;
  bus_lines: string[];
}

interface TrendingRoute {
  origin: string;
  destination: string;
  count: number;
  last_search: string;
}

interface BusSimulation {
  id: string;
  position: [number, number];
  stepIndex: number;
  isMoving: boolean;
}

const RouteMap: React.FC = () => {
  const [route, setRoute] = useState<MapRouteResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [loadTime, setLoadTime] = useState<number>(0);
  const [isDarkMode, setIsDarkMode] = useState(false);
  const [busSimulation, setBusSimulation] = useState<BusSimulation | null>(null);
  const [nextBusMinutes, setNextBusMinutes] = useState<number>(0);
  const [isSearching, setIsSearching] = useState(false);
  const searchTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const [origin, setOrigin] = useState<string>('Setor Bueno');
  const [destination, setDestination] = useState<string>('Campus Samambaia');
  const [searchLoading, setSearchLoading] = useState<boolean>(false);
  const [presets, setPresets] = useState<RoutePreset[]>([]);
  const [trending, setTrending] = useState<TrendingRoute[]>([]);
  const [mapBounds, setMapBounds] = useState<LatLngBounds | null>(null);

  const handleSearch = async () => {
    if (!origin.trim() || !destination.trim()) {
      setError('Por favor, preencha origem e destino');
      return;
    }

    setSearchLoading(true);
    const startTime = performance.now();
    
    try {
      const response = await axios.get<MapRouteResponse>('http://localhost:8080/api/v1/map-view', {
        params: {
          origin: origin.trim(),
          destination: destination.trim()
        }
      });
      setRoute(response.data);
      setError(null);
    } catch (err: any) {
      const errorMessage = err.response?.data?.error || 'Failed to load route data';
      setError(errorMessage);
      console.error('Error fetching route:', err);
    } finally {
      const endTime = performance.now();
      setLoadTime(endTime - startTime);
      setSearchLoading(false);
    }
  };

  const loadPresets = async () => {
    try {
      const response = await axios.get('http://localhost:8080/api/v1/route-presets');
      setPresets(response.data.presets);
    } catch (err) {
      console.error('Error loading presets:', err);
    }
  };

  const loadTrending = async () => {
    try {
      const response = await axios.get('http://localhost:8080/api/v1/trending');
      setTrending(response.data.trending);
    } catch (err) {
      console.error('Error loading trending:', err);
    }
  };

  const handlePreset = async (preset: RoutePreset) => {
    setOrigin(preset.origin);
    setDestination(preset.destination);
    // Trigger search automatically
    setTimeout(() => handleSearch(), 100);
  };

  const handleTrendingRoute = async (trendingRoute: TrendingRoute) => {
    setOrigin(trendingRoute.origin);
    setDestination(trendingRoute.destination);
    // Trigger search automatically (cache hit esperado)
    setTimeout(() => handleSearch(), 100);
  };

  useEffect(() => {
    // Carregar rota inicial, presets e trending
    handleSearch();
    loadPresets();
    loadTrending();
  }, []);

  // Auto-zoom quando a rota muda
  useEffect(() => {
    if (route && route.steps.length > 0) {
      const coordinates = route.steps.map(step => [step.lat, step.lng] as [number, number]);
      const bounds = new LatLngBounds(coordinates);
      setMapBounds(bounds);
    }
  }, [route]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen bg-gray-100">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Loading route...</p>
        </div>
      </div>
    );
  }

  if (error || !route) {
    return (
      <div className="flex items-center justify-center h-screen bg-gray-100">
        <div className="text-center">
          <div className="text-red-600 mb-4">Error: {error}</div>
          <button 
            onClick={() => window.location.reload()} 
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  // Center map on Goiânia (approximate center of the route)
  const center: LatLngExpression = [-16.683, -49.266];
  const routeCoordinates: LatLngExpression[] = route.steps.map(step => [step.lat, step.lng]);

  const getMarkerColor = (step: MapStep) => {
    if (step.name === route.origin.name) return 'bg-green-500';
    if (step.name === route.destination.name) return 'bg-red-500';
    if (step.is_terminal) return 'bg-blue-500';
    return 'bg-gray-500';
  };

  return (
    <div className="h-screen w-full relative">
      {/* Sidebar with trending routes */}
      <div className="absolute top-0 left-0 w-80 bg-white shadow-lg z-10 p-4 h-full overflow-y-auto">
        <h2 className="text-lg font-bold text-gray-800 mb-4">Top Rotas em Goiânia</h2>
        
        {trending.length > 0 && (
          <div className="space-y-2">
            <p className="text-xs text-gray-500 mb-2">Mais buscadas (7 dias)</p>
            {trending.map((trendingRoute, index) => (
              <button
                key={index}
                onClick={() => handleTrendingRoute(trendingRoute)}
                className="w-full text-left p-3 bg-gradient-to-r from-purple-50 to-blue-50 rounded-lg hover:from-purple-100 hover:to-blue-100 transition-colors border border-purple-200"
                disabled={searchLoading}
              >
                <div className="flex justify-between items-start">
                  <div className="flex-1">
                    <p className="font-semibold text-sm text-gray-800">
                      {trendingRoute.origin}
                    </p>
                    <p className="text-xs text-gray-600">
                      {trendingRoute.destination}
                    </p>
                    {trendingRoute.count > 0 && (
                      <p className="text-xs text-purple-600 mt-1">
                        {trendingRoute.count} busca(s)
                      </p>
                    )}
                  </div>
                  <div className="text-xs text-gray-400">
                    #{index + 1}
                  </div>
                </div>
              </button>
            ))}
          </div>
        )}
        
        {trending.length === 0 && (
          <div className="text-center text-gray-500 py-4">
            <p className="text-sm">Carregando rotas populares...</p>
          </div>
        )}
      </div>

      {/* Main content area */}
      <div className="ml-80 h-full">
        {/* Header with search form and route info */}
        <div className="absolute top-0 left-0 right-0 bg-white shadow-md z-10 p-4">
        <div className="max-w-7xl mx-auto">
          <h1 className="text-2xl font-bold text-gray-800 mb-4">
            TranspRota: Busque sua rota
          </h1>
          
          {/* Search Form */}
          <div className="bg-gray-50 rounded-lg p-4 mb-4">
            <div className="flex flex-col sm:flex-row gap-3 mb-3">
              <input
                type="text"
                placeholder="Origem (ex: Setor Bueno)"
                value={origin}
                onChange={(e) => setOrigin(e.target.value)}
                className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                disabled={searchLoading}
              />
              <input
                type="text"
                placeholder="Destino (ex: Campus Samambaia)"
                value={destination}
                onChange={(e) => setDestination(e.target.value)}
                className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                disabled={searchLoading}
              />
              <button
                onClick={handleSearch}
                disabled={searchLoading}
                className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
              >
                {searchLoading ? 'Buscando...' : 'Buscar Rota'}
              </button>
            </div>

            {/* Preset Buttons */}
            {presets.length > 0 && (
              <div className="border-t pt-3">
                <p className="text-sm font-medium text-gray-700 mb-2">Rotas Críticas - Teste Rápido:</p>
                <div className="flex flex-wrap gap-2">
                  {presets.map((preset) => (
                    <button
                      key={preset.id}
                      onClick={() => handlePreset(preset)}
                      className={`px-3 py-1 text-xs rounded-full transition-colors ${
                        preset.complexity === 'Alta'
                          ? 'bg-red-100 text-red-700 hover:bg-red-200'
                          : 'bg-green-100 text-green-700 hover:bg-green-200'
                      }`}
                      disabled={searchLoading}
                    >
                      {preset.name.split(' -> ')[0]}<br/>-> {preset.name.split(' -> ')[1]}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Route Info */}
          {route && (
            <div className="flex flex-wrap gap-4 text-sm text-gray-600">
              <span><strong>Rota:</strong> {route.origin.name} &rarr; {route.destination.name}</span>
              <span><strong>Duração:</strong> {route.total_time_minutes} min</span>
              <span><strong>Linhas:</strong> {route.bus_lines.join(', ')}</span>
              <span><strong>Load Time:</strong> {loadTime.toFixed(2)}ms</span>
              <span><strong>Status:</strong> 
                <span className="text-green-600 font-semibold">Connected</span>
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Map */}
      <div className="h-full pt-24">
        <MapContainer
          center={center}
          zoom={14}
          style={{ height: '100%', width: '100%' }}
        >
          <TileLayer
            attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
            url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
          />
          
          {/* Route polyline */}
          <Polyline
            positions={routeCoordinates}
            color="blue"
            weight={4}
            opacity={0.7}
          />
          
          {/* Markers for each step */}
          {route.steps.map((step, index) => (
            <Marker key={index} position={[step.lat, step.lng]}>
              <Popup>
                <div className="text-sm">
                  <div className="font-semibold">{step.name}</div>
                  {step.is_terminal && (
                    <div className="text-blue-600">Terminal</div>
                  )}
                  {step.is_transfer && (
                    <div className="text-orange-600">Transfer Point</div>
                  )}
                  {step.name === route.origin.name && (
                    <div className="text-green-600">Origin</div>
                  )}
                  {step.name === route.destination.name && (
                    <div className="text-red-600">Destination</div>
                  )}
                </div>
              </Popup>
            </Marker>
          ))}
        </MapContainer>
      </div>

      {/* Legend */}
      <div className="absolute bottom-4 left-4 bg-white rounded-lg shadow-md p-3 z-10">
        <div className="text-xs font-semibold mb-2">Legend</div>
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 bg-green-500 rounded-full"></div>
            <span className="text-xs">Origin</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 bg-red-500 rounded-full"></div>
            <span className="text-xs">Destination</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 bg-blue-500 rounded-full"></div>
            <span className="text-xs">Terminal</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 bg-orange-500 rounded-full"></div>
            <span className="text-xs">Transfer</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default RouteMap;
