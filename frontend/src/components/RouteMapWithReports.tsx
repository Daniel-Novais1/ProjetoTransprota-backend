import React, { useState, useEffect, useRef } from 'react';
import { MapContainer, TileLayer, Polyline, Marker, Popup } from 'react-leaflet';
import { LatLngBounds, DivIcon } from 'leaflet';
import axios from 'axios';
import ReportButton from './ReportButton';

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

interface UserReport {
  id: number;
  tipo_problema: string;
  descricao: string;
  latitude: number;
  longitude: number;
  bus_line?: string;
  user_ip_hash: string;
  trust_score: number;
  status: string;
  created_at: string;
}

interface HeatmapData {
  bus_line: string;
  tipo_problema: string;
  report_count: number;
  avg_trust_score: number;
  centroid_lat: number;
  centroid_lng: number;
  severity: string;
}

interface BusSimulation {
  id: string;
  position: [number, number];
  stepIndex: number;
  isMoving: boolean;
}

const RouteMapWithReports: React.FC = () => {
  const [route, setRoute] = useState<MapRouteResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loadTime, setLoadTime] = useState<number>(0);
  const [isDarkMode, setIsDarkMode] = useState(false);
  const [busSimulation, setBusSimulation] = useState<BusSimulation | null>(null);
  const [nextBusMinutes, setNextBusMinutes] = useState<number>(0);
  const [isSearching, setIsSearching] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const [isFormExpanded, setIsFormExpanded] = useState(false);
  const searchTimeoutRef = useRef<any>(null);
  const [origin, setOrigin] = useState<string>('Setor Bueno');
  const [destination, setDestination] = useState<string>('Campus Samambaia');
  const [searchLoading, setSearchLoading] = useState<boolean>(false);
  const [mapBounds, setMapBounds] = useState<LatLngBounds | null>(null);
  
  // Estados para denúncias
  const [reports, setReports] = useState<UserReport[]>([]);
  const [heatmapData, setHeatmapData] = useState<HeatmapData[]>([]);
  const [currentLocation, setCurrentLocation] = useState<{ lat: number; lng: number } | null>(null);

  // Detectar dispositivo mobile
  useEffect(() => {
    const checkMobile = () => {
      const isMobileDevice = window.innerWidth <= 768;
      setIsMobile(isMobileDevice);
    };

    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, []);

  // Dark Mode automático
  useEffect(() => {
    const checkDarkMode = () => {
      const goianiaTime = new Date().toLocaleTimeString('en-US', { timeZone: 'America/Sao_Paulo' });
      const hour = parseInt(goianiaTime.split(':')[0]);
      const shouldBeDark = hour >= 18 || hour < 6;
      setIsDarkMode(shouldBeDark);
    };

    checkDarkMode();
    const interval = setInterval(checkDarkMode, 60000);
    return () => clearInterval(interval);
  }, []);

  // Calcular próximo ônibus
  useEffect(() => {
    const calculateNextBus = () => {
      const now = new Date();
      const hour = now.getHours();
      const interval = (hour >= 7 && hour <= 9) || (hour >= 17 && hour <= 19) ? 15 : 20;
      const currentMinutes = hour * 60 + now.getMinutes();
      const nextBusMinutes = Math.ceil(currentMinutes / interval) * interval;
      const minutesUntilNext = nextBusMinutes - currentMinutes;
      setNextBusMinutes(minutesUntilNext);
    };

    calculateNextBus();
    const interval = setInterval(calculateNextBus, 30000);
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
          currentStep = 0;
        }

        const step = route.steps[currentStep];
        const nextStep = route.steps[currentStep + 1];
        
        if (nextStep) {
          const progress = (Date.now() % 3000) / 3000;
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

  // Carregar denúncias recentes e heatmap
  useEffect(() => {
    const loadReportsData = async () => {
      try {
        // Carregar denúncias recentes
        const reportsResponse = await axios.get('http://localhost:8080/api/v1/reports/recent');
        setReports(reportsResponse.data.reports || []);

        // Carregar heatmap data
        const heatmapResponse = await axios.get('http://localhost:8080/api/v1/reports/heatmap');
        setHeatmapData(heatmapResponse.data.heatmap || []);
      } catch (err) {
        console.error('Erro ao carregar denúncias:', err);
      }
    };

    loadReportsData();
    const interval = setInterval(loadReportsData, 60000); // Atualizar a cada minuto
    return () => clearInterval(interval);
  }, []);

  // Obter localização atual
  useEffect(() => {
    if (navigator.geolocation) {
      navigator.geolocation.getCurrentPosition(
        (position) => {
          setCurrentLocation({
            lat: position.coords.latitude,
            lng: position.coords.longitude
          });
        },
        (error) => {
          console.log('Geolocalização não disponível:', error);
        }
      );
    }
  }, []);

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
        
        // Auto-zoom e colapsar formulário em mobile
        if (response.data.steps && response.data.steps.length > 0) {
          const bounds = new (window as any).L.LatLngBounds([
            [response.data.origin.lat, response.data.origin.lng],
            [response.data.destination.lat, response.data.destination.lng]
          ]);
          setMapBounds(bounds);
          if (isMobile) {
            setIsFormExpanded(false);
          }
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

  // Callback para quando uma denúncia é enviada
  const handleReportSubmit = (report: any) => {
    // Recarregar dados de denúncias
    setTimeout(() => {
      loadReportsData();
    }, 1000);
  };

  const loadReportsData = async () => {
    try {
      const reportsResponse = await axios.get('http://localhost:8080/api/v1/reports/recent');
      setReports(reportsResponse.data.reports || []);

      const heatmapResponse = await axios.get('http://localhost:8080/api/v1/reports/heatmap');
      setHeatmapData(heatmapResponse.data.heatmap || []);
    } catch (err) {
      console.error('Erro ao recarregar denúncias:', err);
    }
  };

  // Ícones customizados
  const createBusIcon = () => {
    const size = isMobile ? 20 : 24;
    return new DivIcon({
      html: `
        <div style="
          background: ${isDarkMode ? '#1f2937' : '#3b82f6'};
          border: 2px solid ${isDarkMode ? '#60a5fa' : '#1e40af'};
          border-radius: 50%;
          width: ${size}px;
          height: ${size}px;
          display: flex;
          align-items: center;
          justify-content: center;
          box-shadow: 0 2px 8px rgba(0,0,0,0.3);
          animation: pulse 2s infinite;
        ">
          <span style="color: white; font-size: ${size * 0.4}px; font-weight: bold;">BUS</span>
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
      iconSize: [size, size],
      iconAnchor: [size/2, size/2]
    });
  };

  const createTerminalIcon = (isTransfer: boolean = false) => {
    const size = isMobile ? 28 : 32;
    return new DivIcon({
      html: `
        <div style="
          background: ${isTransfer ? '#f59e0b' : '#10b981'};
          border: 2px solid ${isTransfer ? '#d97706' : '#059669'};
          border-radius: 8px;
          width: ${size}px;
          height: ${size}px;
          display: flex;
          align-items: center;
          justify-content: center;
          box-shadow: 0 2px 8px rgba(0,0,0,0.3);
        ">
          <span style="color: white; font-size: ${size * 0.35}px; font-weight: bold;">
            ${isTransfer ? 'T' : 'B'}
          </span>
        </div>
      `,
      className: 'custom-terminal-icon',
      iconSize: [size, size],
      iconAnchor: [size/2, size/2]
    });
  };

  const createReportIcon = (tipoProblema: string) => {
    const colors = {
      'Lotado': '#f59e0b',
      'Atrasado': '#3b82f6',
      'Perigo': '#ef4444'
    };
    const icons = {
      'Lotado': 'crowd',
      'Atrasado': 'clock',
      'Perigo': 'warning'
    };

    const size = isMobile ? 24 : 28;
    return new DivIcon({
      html: `
        <div style="
          background: ${colors[tipoProblema as keyof typeof colors] || '#6b7280'};
          border: 2px solid white;
          border-radius: 50%;
          width: ${size}px;
          height: ${size}px;
          display: flex;
          align-items: center;
          justify-content: center;
          box-shadow: 0 2px 8px rgba(0,0,0,0.4);
          animation: blink 2s infinite;
        ">
          <span style="color: white; font-size: ${size * 0.3}px; font-weight: bold;">
            ${icons[tipoProblema as keyof typeof icons] || '!'}
          </span>
        </div>
        <style>
          @keyframes blink {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.6; }
          }
        </style>
      `,
      className: 'custom-report-icon',
      iconSize: [size, size],
      iconAnchor: [size/2, size/2]
    });
  };

  // Calcular cor da rota baseada no heatmap
  const getRouteColor = () => {
    if (!route || !heatmapData.length) return isDarkMode ? '#60a5fa' : '#3b82f6';
    
    // Verificar se há denúncias para esta rota
    const routeReports = heatmapData.filter(data => 
      route.bus_lines.includes(data.bus_line)
    );

    if (routeReports.length === 0) return isDarkMode ? '#60a5fa' : '#3b82f6';

    // Priorizar severidade
    const hasHighSeverity = routeReports.some(r => r.severity === 'alta');
    const hasMediumSeverity = routeReports.some(r => r.severity === 'media');

    if (hasHighSeverity) return '#ef4444'; // Vermelho para alta severidade
    if (hasMediumSeverity) return '#f59e0b'; // Laranja para média severidade
    return '#3b82f6'; // Azul para baixa severidade
  };

  const getTileUrl = () => {
    return isDarkMode 
      ? 'https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png'
      : 'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png';
  };

  // Estilos responsivos
  const containerStyle = {
    height: '100vh',
    width: '100%',
    backgroundColor: isDarkMode ? '#111827' : '#f3f4f6',
    transition: 'all 0.3s ease'
  };

  const getFormStyle = () => {
    const baseStyle = {
      position: 'absolute' as const,
      left: '10px',
      right: '10px',
      zIndex: 1000,
      backgroundColor: isDarkMode ? 'rgba(17, 24, 39, 0.95)' : 'rgba(255, 255, 255, 0.95)',
      backdropFilter: 'blur(10px)',
      borderRadius: isMobile ? '8px' : '12px',
      boxShadow: '0 4px 20px rgba(0,0,0,0.15)',
      border: `1px solid ${isDarkMode ? '#374151' : '#e5e7eb'}`,
      transition: 'all 0.3s ease',
      overflow: 'hidden'
    };

    if (isMobile) {
      return {
        ...baseStyle,
        top: isFormExpanded ? '10px' : '10px',
        maxHeight: isFormExpanded ? '400px' : '80px',
        padding: isFormExpanded ? '16px' : '12px'
      };
    } else {
      return {
        ...baseStyle,
        top: '20px',
        padding: '16px',
        maxHeight: 'none'
      };
    }
  };

  return (
    <div style={containerStyle}>
      {/* Formulário responsivo */}
      <div style={getFormStyle()}>
        {/* Toggle button para mobile */}
        {isMobile && (
          <button
            onClick={() => setIsFormExpanded(!isFormExpanded)}
            style={{
              position: 'absolute' as const,
              top: isFormExpanded ? 'auto' : '50%',
              right: '10px',
              transform: isFormExpanded ? 'translateY(0)' : 'translateY(-50%)',
              backgroundColor: isDarkMode ? '#374151' : '#e5e7eb',
              color: isDarkMode ? '#f3f4f6' : '#111827',
              border: 'none',
              borderRadius: '4px',
              padding: '4px 8px',
              fontSize: '12px',
              cursor: 'pointer',
              zIndex: 1001
            }}
          >
            {isFormExpanded ? 'Collapse' : 'Expand'}
          </button>
        )}

        <h3 style={{ 
          color: isDarkMode ? '#f3f4f6' : '#111827', 
          margin: isMobile ? (isFormExpanded ? '0 0 12px 0' : '0') : '0 0 16px 0',
          fontSize: isMobile ? '16px' : '18px',
          fontWeight: '700',
          textAlign: isMobile ? 'center' : 'left'
        }}>
          {isDarkMode ? 'TranspRota Noturno' : 'TranspRota Diurno'}
        </h3>
        
        {/* Campos do formulário - visíveis apenas quando expandido em mobile */}
        {(isFormExpanded || !isMobile) && (
          <>
            <input
              type="text"
              placeholder="Origem"
              value={origin}
              onChange={(e) => setOrigin(e.target.value)}
              style={{
                backgroundColor: isDarkMode ? '#1f2937' : '#ffffff',
                color: isDarkMode ? '#f3f4f6' : '#111827',
                border: `1px solid ${isDarkMode ? '#374151' : '#d1d5db'}`,
                borderRadius: '6px',
                padding: isMobile ? '10px 12px' : '12px 16px',
                fontSize: '14px',
                width: '100%',
                marginBottom: '8px',
                transition: 'all 0.2s ease',
                boxSizing: 'border-box' as const
              }}
            />
            
            <input
              type="text"
              placeholder="Destino"
              value={destination}
              onChange={(e) => setDestination(e.target.value)}
              style={{
                backgroundColor: isDarkMode ? '#1f2937' : '#ffffff',
                color: isDarkMode ? '#f3f4f6' : '#111827',
                border: `1px solid ${isDarkMode ? '#374151' : '#d1d5db'}`,
                borderRadius: '6px',
                padding: isMobile ? '10px 12px' : '12px 16px',
                fontSize: '14px',
                width: '100%',
                marginBottom: '8px',
                transition: 'all 0.2s ease',
                boxSizing: 'border-box' as const
              }}
            />
            
            <button
              onClick={handleSearch}
              disabled={isSearching}
              style={{
                backgroundColor: isSearching ? '#6b7280' : '#3b82f6',
                color: 'white',
                border: 'none',
                borderRadius: '6px',
                padding: isMobile ? '10px 16px' : '12px 24px',
                fontSize: '14px',
                fontWeight: '600',
                cursor: isSearching ? 'not-allowed' : 'pointer',
                width: '100%',
                transition: 'all 0.2s ease',
                opacity: isSearching ? 0.7 : 1,
                marginTop: '4px'
              }}
            >
              {isSearching ? 'Buscando...' : 'Buscar Rota'}
            </button>

            {nextBusMinutes > 0 && (
              <div style={{
                backgroundColor: isDarkMode ? '#065f46' : '#d1fae5',
                color: isDarkMode ? '#6ee7b7' : '#065f46',
                padding: isMobile ? '6px 10px' : '8px 12px',
                borderRadius: '4px',
                fontSize: isMobile ? '11px' : '12px',
                fontWeight: '600',
                marginTop: '8px',
                textAlign: 'center' as const
              }}>
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
          </>
        )}
      </div>

      {/* Botão de Report */}
      <ReportButton
        onReportSubmit={handleReportSubmit}
        currentLocation={currentLocation}
        currentBusLine={route?.bus_lines[0]}
        isDarkMode={isDarkMode}
      />

      {/* Mapa */}
      {route && (
        <MapContainer
          bounds={mapBounds || undefined}
          style={{ height: '100%', width: '100%' }}
          zoom={13}
        >
          <TileLayer
            url={getTileUrl()}
            attribution="&copy; OpenStreetMap contributors"
          />

          {/* Rota polyline com cor baseada no heatmap */}
          {route.steps && route.steps.length > 1 && (
            <Polyline
              positions={route.steps.map(step => [step.lat, step.lng])}
              color={getRouteColor()}
              weight={isMobile ? 3 : 4}
              opacity={0.8}
            />
          )}

          {/* Marcadores de denúncias recentes */}
          {reports.map((report) => (
            <Marker
              key={`report-${report.id}`}
              position={[report.latitude, report.longitude]}
              icon={createReportIcon(report.tipo_problema)}
            >
              <Popup>
                <div style={{ textAlign: 'center', fontSize: isMobile ? '12px' : '14px' }}>
                  <strong>{report.tipo_problema}</strong>
                  <br />
                  <small>{report.descricao || 'Sem descrição'}</small>
                  <br />
                  <small>Confiança: {(report.trust_score * 100).toFixed(0)}%</small>
                </div>
              </Popup>
            </Marker>
          ))}

          {/* Marcadores de terminais */}
          {route.steps && route.steps.map((step, index) => (
            <Marker
              key={`terminal-${index}`}
              position={[step.lat, step.lng]}
              icon={createTerminalIcon(step.is_transfer)}
            >
              <Popup>
                <div style={{ textAlign: 'center', fontSize: isMobile ? '12px' : '14px' }}>
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
                <div style={{ textAlign: 'center', fontSize: isMobile ? '12px' : '14px' }}>
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
              <div style={{ fontSize: isMobile ? '12px' : '14px' }}>
                <strong>Origem:</strong> {route.origin.name}
              </div>
            </Popup>
          </Marker>

          <Marker position={[route.destination.lat, route.destination.lng]}>
            <Popup>
              <div style={{ fontSize: isMobile ? '12px' : '14px' }}>
                <strong>Destino:</strong> {route.destination.name}
              </div>
            </Popup>
          </Marker>
        </MapContainer>
      )}
    </div>
  );
};

export default RouteMapWithReports;
