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

const RouteMapEnhanced: React.FC = () => {
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

  // Dark Mode automático baseado no horário de Goiânia
  useEffect(() => {
    const checkDarkMode = () => {
      const currentHour = new Date().getHours();
      const goianiaTime = new Date().toLocaleTimeString('en-US', { timeZone: 'America/Sao_Paulo' });
      const hour = parseInt(goianiaTime.split(':')[0]);
      
      // Dark mode das 18h às 6h (horário de Goiânia)
      const shouldBeDark = hour >= 18 || hour < 6;
      setIsDarkMode(shouldBeDark);
    };

    checkDarkMode();
    const interval = setInterval(checkDarkMode, 60000); // Verificar a cada minuto
    return () => clearInterval(interval);
  }, []);

  // Calcular próximo ônibus baseado no horário atual
  useEffect(() => {
    const calculateNextBus = () => {
      const now = new Date();
      const currentMinutes = now.getHours() * 60 + now.getMinutes();
      
      // Intervalos fixos de 15-20 minutos dependendo do horário
      const hour = now.getHours();
      const interval = (hour >= 7 && hour <= 9) || (hour >= 17 && hour <= 19) ? 15 : 20;
      
      // Próximo ônibus no múltiplo mais próximo do intervalo
      const nextBusMinutes = Math.ceil(currentMinutes / interval) * interval;
      const minutesUntilNext = nextBusMinutes - currentMinutes;
      
      setNextBusMinutes(minutesUntilNext);
    };

    calculateNextBus();
    const interval = setInterval(calculateNextBus, 30000); // Atualizar a cada 30 segundos
    return () => clearInterval(interval);
  }, []);

  // Simulação de movimento do ônibus
  useEffect(() => {
    if (!route || !route.steps || route.steps.length < 2) return;

    const startSimulation = () => {
      let currentStep = 0;
      const simulationId = `bus-${Date.now()}`;
      
      const moveBus = () => {
        if (currentStep >= route.steps.length - 1) {
          currentStep = 0; // Reiniciar ciclo
        }

        const step = route.steps[currentStep];
        const nextStep = route.steps[currentStep + 1];
        
        if (nextStep) {
          // Interpolar posição entre os pontos
          const progress = (Date.now() % 3000) / 3000; // 3 segundos por passo
          const lat = step.lat + (nextStep.lat - step.lat) * progress;
          const lng = step.lng + (nextStep.lng - step.lng) * progress;
          
          setBusSimulation({
            id: simulationId,
            position: [lat, lng],
            stepIndex: currentStep,
            isMoving: true
          });

          if (progress >= 0.95) {
            currentStep++;
          }
        }
      };

      const interval = setInterval(moveBus, 100);
      return () => clearInterval(interval);
    };

    const cleanup = startSimulation();
    return cleanup;
  }, [route]);

  // Debounce para rate limiting visual
  const debouncedSearch = (searchFunction: () => void, delay: number = 800) => {
    if (searchTimeoutRef.current) {
      clearTimeout(searchTimeoutRef.current);
    }
    
    setIsSearching(true);
    searchTimeoutRef.current = setTimeout(() => {
      searchFunction();
      setIsSearching(false);
    }, delay);
  };

  const handleSearch = async () => {
    if (!origin.trim() || !destination.trim()) {
      setError('Por favor, preencha origem e destino');
      return;
    }

    const searchFunction = async () => {
      try {
        setSearchLoading(true);
        setError(null);
        
        const startTime = performance.now();
        const response = await axios.get(`http://localhost:8080/api/v1/map-view?origin=${encodeURIComponent(origin)}&destination=${encodeURIComponent(destination)}`);
        const endTime = performance.now();
        
        setLoadTime(endTime - startTime);
        setRoute(response.data);
        
        // Auto-zoom para mostrar a rota completa
        if (response.data.steps && response.data.steps.length > 0) {
          const bounds = new (window as any).L.LatLngBounds([
            [response.data.origin.lat, response.data.origin.lng],
            [response.data.destination.lat, response.data.destination.lng]
          ]);
          setMapBounds(bounds);
        }
      } catch (err) {
        setError('Erro ao buscar rota. Tente novamente.');
        console.error('Search error:', err);
      } finally {
        setSearchLoading(false);
      }
    };

    debouncedSearch(searchFunction);
  };

  // Ícones customizados para marcadores modernos
  const createBusIcon = () => {
    return new DivIcon({
      html: `
        <div style="
          background: ${isDarkMode ? '#1f2937' : '#3b82f6'};
          border: 2px solid ${isDarkMode ? '#60a5fa' : '#1e40af'};
          border-radius: 50%;
          width: 24px;
          height: 24px;
          display: flex;
          align-items: center;
          justify-content: center;
          box-shadow: 0 2px 8px rgba(0,0,0,0.3);
          animation: pulse 2s infinite;
        ">
          <span style="color: white; font-size: 12px; font-weight: bold;">BUS</span>
        </div>
        <style>
          @keyframes pulse {
            0% { transform: scale(1); }
            50% { transform: scale(1.1); }
            100% { transform: scale(1); }
          }
        </style>
      `,
      className: 'custom-bus-icon',
      iconSize: [24, 24],
      iconAnchor: [12, 12]
    });
  };

  const createTerminalIcon = (isTransfer: boolean = false) => {
    return new DivIcon({
      html: `
        <div style="
          background: ${isTransfer ? '#f59e0b' : '#10b981'};
          border: 2px solid ${isTransfer ? '#d97706' : '#059669'};
          border-radius: 8px;
          width: 32px;
          height: 32px;
          display: flex;
          align-items: center;
          justify-content: center;
          box-shadow: 0 2px 8px rgba(0,0,0,0.3);
        ">
          <span style="color: white; font-size: 14px; font-weight: bold;">
            ${isTransfer ? 'T' : 'B'}
          </span>
        </div>
      `,
      className: 'custom-terminal-icon',
      iconSize: [32, 32],
      iconAnchor: [16, 16]
    });
  };

  const getTileUrl = () => {
    return isDarkMode 
      ? 'https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png'
      : 'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png';
  };

  const getTileAttribution = () => {
    return isDarkMode
      ? '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>'
      : '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors';
  };

  // Estilos dinâmicos baseados no tema
  const containerStyle = {
    height: '100vh',
    width: '100%',
    backgroundColor: isDarkMode ? '#111827' : '#f3f4f6',
    transition: 'all 0.3s ease'
  };

  const formStyle = {
    position: 'absolute' as const,
    top: '20px',
    left: '20px',
    right: '20px',
    zIndex: 1000,
    backgroundColor: isDarkMode ? 'rgba(17, 24, 39, 0.95)' : 'rgba(255, 255, 255, 0.95)',
    backdropFilter: 'blur(10px)',
    borderRadius: '12px',
    padding: '16px',
    boxShadow: '0 4px 20px rgba(0,0,0,0.15)',
    border: `1px solid ${isDarkMode ? '#374151' : '#e5e7eb'}`
  };

  const inputStyle = {
    backgroundColor: isDarkMode ? '#1f2937' : '#ffffff',
    color: isDarkMode ? '#f3f4f6' : '#111827',
    border: `1px solid ${isDarkMode ? '#374151' : '#d1d5db'}`,
    borderRadius: '8px',
    padding: '12px 16px',
    fontSize: '14px',
    width: '100%',
    marginBottom: '12px',
    transition: 'all 0.2s ease'
  };

  const buttonStyle = {
    backgroundColor: isSearching ? '#6b7280' : '#3b82f6',
    color: 'white',
    border: 'none',
    borderRadius: '8px',
    padding: '12px 24px',
    fontSize: '14px',
    fontWeight: '600',
    cursor: isSearching ? 'not-allowed' : 'pointer',
    width: '100%',
    transition: 'all 0.2s ease',
    opacity: isSearching ? 0.7 : 1
  };

  const nextBusStyle = {
    backgroundColor: isDarkMode ? '#065f46' : '#d1fae5',
    color: isDarkMode ? '#6ee7b7' : '#065f46',
    padding: '8px 12px',
    borderRadius: '6px',
    fontSize: '12px',
    fontWeight: '600',
    marginTop: '8px',
    textAlign: 'center' as const
  };

  return (
    <div style={containerStyle}>
      <div style={formStyle}>
        <h3 style={{ 
          color: isDarkMode ? '#f3f4f6' : '#111827', 
          margin: '0 0 16px 0',
          fontSize: '18px',
          fontWeight: '700'
        }}>
          {isDarkMode ? 'TranspRota Noturno' : 'TranspRota Diurno'}
        </h3>
        
        <input
          type="text"
          placeholder="Origem"
          value={origin}
          onChange={(e) => setOrigin(e.target.value)}
          style={inputStyle}
        />
        
        <input
          type="text"
          placeholder="Destino"
          value={destination}
          onChange={(e) => setDestination(e.target.value)}
          style={inputStyle}
        />
        
        <button
          onClick={handleSearch}
          disabled={isSearching}
          style={buttonStyle}
        >
          {isSearching ? 'Buscando...' : 'Buscar Rota'}
        </button>

        {nextBusMinutes > 0 && (
          <div style={nextBusStyle}>
            Próximo ônibus em {nextBusMinutes} min
          </div>
        )}

        {searchLoading && (
          <div style={{ 
            textAlign: 'center', 
            marginTop: '8px', 
            fontSize: '12px',
            color: isDarkMode ? '#9ca3af' : '#6b7280'
          }}>
            Carregando rota...
          </div>
        )}

        {error && (
          <div style={{ 
            color: '#ef4444', 
            marginTop: '8px', 
            fontSize: '12px',
            textAlign: 'center'
          }}>
            {error}
          </div>
        )}

        {loadTime > 0 && (
          <div style={{ 
            fontSize: '10px', 
            marginTop: '8px',
            color: isDarkMode ? '#6b7280' : '#9ca3af',
            textAlign: 'center'
          }}>
            Tempo: {loadTime.toFixed(0)}ms
          </div>
        )}
      </div>

      {route && (
        <MapContainer
          bounds={mapBounds || undefined}
          style={{ height: '100%', width: '100%' }}
          zoom={13}
        >
          <TileLayer
            url={getTileUrl()}
            attribution={getTileAttribution()}
          />

          {/* Rota polyline */}
          {route.steps && route.steps.length > 1 && (
            <Polyline
              positions={route.steps.map(step => [step.lat, step.lng])}
              color={isDarkMode ? '#60a5fa' : '#3b82f6'}
              weight={4}
              opacity={0.8}
            />
          )}

          {/* Marcadores de terminais */}
          {route.steps && route.steps.map((step, index) => (
            <Marker
              key={`terminal-${index}`}
              position={[step.lat, step.lng]}
              icon={createTerminalIcon(step.is_transfer)}
            >
              <Popup>
                <div style={{ textAlign: 'center' }}>
                  <strong>{step.name}</strong>
                  <br />
                  <small>{step.is_transfer ? 'Ponto de Integração' : 'Parada'}</small>
                </div>
              </Popup>
            </Marker>
          ))}

          {/* Simulação de ônibus em movimento */}
          {busSimulation && (
            <Marker
              position={busSimulation.position}
              icon={createBusIcon()}
            >
              <Popup>
                <div style={{ textAlign: 'center' }}>
                  <strong>Ônibus em Movimento</strong>
                  <br />
                  <small>Simulação ao vivo</small>
                </div>
              </Popup>
            </Marker>
          )}

          {/* Marcadores de origem e destino */}
          <Marker position={[route.origin.lat, route.origin.lng]}>
            <Popup>
              <strong>Origem:</strong> {route.origin.name}
            </Popup>
          </Marker>

          <Marker position={[route.destination.lat, route.destination.lng]}>
            <Popup>
              <strong>Destino:</strong> {route.destination.name}
            </Popup>
          </Marker>
        </MapContainer>
      )}
    </div>
  );
};

export default RouteMapEnhanced;
