import React, { useState, useEffect } from 'react';
import axios from 'axios';

interface SystemHealth {
  status: 'online' | 'degraded' | 'offline';
  database: string;
  redis: string;
  uptime: number;
  timestamp: string;
}

interface TrendingRoute {
  origin: string;
  destination: string;
  count: number;
  last_search: string;
  crisis_level?: 'normal' | 'warning' | 'critical';
  report_count?: number;
  crisis_score?: number;
}

interface ReportData {
  tipo_problema: string;
  count: number;
  severity: 'low' | 'medium' | 'high';
}

interface DashboardData {
  system_health: SystemHealth;
  trending_routes: TrendingRoute[];
  recent_reports: ReportData[];
  metrics: {
    total_requests: number;
    error_rate: number;
    active_users: number;
    cache_hit_rate: number;
  };
}

const AdminDashboard: React.FC = () => {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isDarkMode, setIsDarkMode] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [refreshInterval, setRefreshInterval] = useState(5000);

  // Dark mode automático baseado no horário
  useEffect(() => {
    const checkDarkMode = () => {
      const hour = new Date().getHours();
      const shouldBeDark = hour >= 18 || hour < 6;
      setIsDarkMode(shouldBeDark);
    };

    checkDarkMode();
    const interval = setInterval(checkDarkMode, 60000);
    return () => clearInterval(interval);
  }, []);

  // Carregar dados do dashboard
  const fetchDashboardData = async () => {
    try {
      const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080';
      const response = await axios.get(`${apiUrl}/api/v1/admin/dashboard`);
      setData(response.data);
      setError(null);
    } catch (err: any) {
      console.error('Erro ao carregar dashboard:', err);
      setError('Erro ao carregar dados do dashboard');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDashboardData();
    
    if (autoRefresh) {
      const interval = setInterval(fetchDashboardData, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [autoRefresh, refreshInterval]);

  // Exportar CSV
  const exportCSV = async () => {
    try {aiUrlmprtma.env.VITE_API_URL || ';
      const response = await axios.get(`${apiUrl}`
      const response = await axios.get('http://localhost:8080/api/v1/admin/export/csv');
      
      // Criar blob e download
      const blob = new Blob([response.data], { type: 'text/csv' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `transprota-dashboard-${new Date().toISOString().split('T')[0]}.csv`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (err: any) {
      console.error('Erro ao exportar CSV:', err);
      setError('Erro ao exportar CSV');
    }
  };

  // Estilos dinâmicos baseados no tema e status
  const getHealthColor = (status: string) => {
    switch (status) {
      case 'online':
        return isDarkMode ? '#10b981' : '#059669'; // Verde Esmeralda
      case 'degraded':
        return isDarkMode ? '#f97316' : '#ea580c'; // Laranja Neon
      case 'offline':
        return isDarkMode ? '#ef4444' : '#dc2626'; // Vermelho
      default:
        return '#6b7280';
    }
  };

  const getCrisisColor = (level?: string) => {
    switch (level) {
      case 'critical':
        return isDarkMode ? '#ef4444' : '#dc2626';
      case 'warning':
        return isDarkMode ? '#f97316' : '#ea580c';
      case 'normal':
        return isDarkMode ? '#10b981' : '#059669';
      default:
        return '#6b7280';
    }
  };

  const containerStyle = {
    minHeight: '100vh',
    backgroundColor: isDarkMode ? '#111827' : '#f8fafc',
    color: isDarkMode ? '#f3f4f6' : '#111827',
    padding: '20px',
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
  };

  const headerStyle = {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '30px',
    padding: '20px',
    backgroundColor: isDarkMode ? '#1f2937' : '#ffffff',
    borderRadius: '12px',
    boxShadow: '0 4px 20px rgba(0,0,0,0.1)'
  };

  const cardStyle = {
    backgroundColor: isDarkMode ? '#1f2937' : '#ffffff',
    borderRadius: '12px',
    padding: '20px',
    boxShadow: '0 4px 20px rgba(0,0,0,0.1)',
    marginBottom: '20px',
    border: `1px solid ${isDarkMode ? '#374151' : '#e5e7eb'}`
  };

  const healthIndicatorStyle = (status: string) => ({
    display: 'inline-block',
    width: '12px',
    height: '12px',
    borderRadius: '50%',
    backgroundColor: getHealthColor(status),
    marginRight: '8px',
    animation: status === 'online' ? 'pulse 2s infinite' : 'none'
  });

  const crisisBadgeStyle = (level?: string) => ({
    display: 'inline-block',
    padding: '4px 8px',
    borderRadius: '4px',
    fontSize: '11px',
    fontWeight: '600',
    backgroundColor: getCrisisColor(level),
    color: 'white',
    marginLeft: '8px'
  });

  if (loading) {
    return (
      <div style={containerStyle}>
        <div style={{ textAlign: 'center', marginTop: '100px' }}>
          <div style={{ fontSize: '18px', fontWeight: '600' }}>
            Carregando Dashboard...
          </div>
        </div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div style={containerStyle}>
        <div style={{ textAlign: 'center', marginTop: '100px' }}>
          <div style={{ color: '#ef4444', fontSize: '18px', fontWeight: '600' }}>
            {error || 'Dados não disponíveis'}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div style={containerStyle}>
      <style>
        {`
          @keyframes pulse {
            0% { transform: scale(1); opacity: 1; }
            50% { transform: scale(1.1); opacity: 0.8; }
            100% { transform: scale(1); opacity: 1; }
          }
          @keyframes slideIn {
            from { transform: translateY(20px); opacity: 0; }
            to { transform: translateY(0); opacity: 1; }
          }
          .crisis-row {
            animation: slideIn 0.5s ease-out;
            border-left: 4px solid;
            padding-left: 12px;
            margin-bottom: 8px;
          }
        `}
      </style>

      {/* Header */}
      <div style={headerStyle}>
        <div>
          <h1 style={{ fontSize: '28px', fontWeight: '700', margin: 0 }}>
            TranspRota Admin Dashboard
          </h1>
          <p style={{ color: isDarkMode ? '#9ca3af' : '#6b7280', margin: '4px 0 0 0' }}>
            Monitoramento em tempo real do sistema
          </p>
        </div>
        
        <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
          <button
            onClick={() => setAutoRefresh(!autoRefresh)}
            style={{
              padding: '8px 16px',
              backgroundColor: autoRefresh ? getHealthColor('online') : '#6b7280',
              color: 'white',
              border: 'none',
              borderRadius: '6px',
              fontSize: '14px',
              fontWeight: '600',
              cursor: 'pointer'
            }}
          >
            {autoRefresh ? 'Auto-Refresh ON' : 'Auto-Refresh OFF'}
          </button>
          
          <button
            onClick={exportCSV}
            style={{
              padding: '8px 16px',
              backgroundColor: isDarkMode ? '#374151' : '#e5e7eb',
              color: isDarkMode ? '#f3f4f6' : '#111827',
              border: `1px solid ${isDarkMode ? '#4b5563' : '#d1d5db'}`,
              borderRadius: '6px',
              fontSize: '14px',
              fontWeight: '600',
              cursor: 'pointer'
            }}
          >
            Exportar CSV
          </button>
        </div>
      </div>

      {/* System Health */}
      <div style={cardStyle}>
        <h2 style={{ fontSize: '20px', fontWeight: '700', margin: '0 0 16px 0' }}>
          Saúde do Sistema
        </h2>
        
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '16px' }}>
          <div style={{ textAlign: 'center' }}>
            <div style={healthIndicatorStyle(data.system_health.status)} />
            <div style={{ fontSize: '16px', fontWeight: '600' }}>
              {data.system_health.status.toUpperCase()}
            </div>
            <div style={{ fontSize: '12px', color: isDarkMode ? '#9ca3af' : '#6b7280' }}>
              Status Geral
            </div>
          </div>
          
          <div style={{ textAlign: 'center' }}>
            <div style={healthIndicatorStyle(data.system_health.database === 'ok' ? 'online' : 'offline')} />
            <div style={{ fontSize: '16px', fontWeight: '600' }}>
              {data.system_health.database.toUpperCase()}
            </div>
            <div style={{ fontSize: '12px', color: isDarkMode ? '#9ca3af' : '#6b7280' }}>
              PostgreSQL
            </div>
          </div>
          
          <div style={{ textAlign: 'center' }}>
            <div style={healthIndicatorStyle(data.system_health.redis === 'ok' ? 'online' : 'offline')} />
            <div style={{ fontSize: '16px', fontWeight: '600' }}>
              {data.system_health.redis.toUpperCase()}
            </div>
            <div style={{ fontSize: '12px', color: isDarkMode ? '#9ca3af' : '#6b7280' }}>
              Redis Cache
            </div>
          </div>
          
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: '16px', fontWeight: '600' }}>
              {Math.floor(data.system_health.uptime / 60)}m
            </div>
            <div style={{ fontSize: '12px', color: isDarkMode ? '#9ca3af' : '#6b7280' }}>
              Uptime
            </div>
          </div>
        </div>
      </div>

      {/* Métricas */}
      <div style={cardStyle}>
        <h2 style={{ fontSize: '20px', fontWeight: '700', margin: '0 0 16px 0' }}>
          Métricas do Sistema
        </h2>
        
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: '16px' }}>
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: '24px', fontWeight: '700', color: getHealthColor('online') }}>
              {data.metrics.total_requests.toLocaleString()}
            </div>
            <div style={{ fontSize: '12px', color: isDarkMode ? '#9ca3af' : '#6b7280' }}>
              Total de Requisições
            </div>
          </div>
          
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: '24px', fontWeight: '700', color: data.metrics.error_rate > 5 ? getHealthColor('offline') : getHealthColor('online') }}>
              {data.metrics.error_rate.toFixed(1)}%
            </div>
            <div style={{ fontSize: '12px', color: isDarkMode ? '#9ca3af' : '#6b7280' }}>
              Taxa de Erro
            </div>
          </div>
          
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: '24px', fontWeight: '700', color: getHealthColor('online') }}>
              {data.metrics.active_users}
            </div>
            <div style={{ fontSize: '12px', color: isDarkMode ? '#9ca3af' : '#6b7280' }}>
              Usuários Ativos
            </div>
          </div>
          
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: '24px', fontWeight: '700', color: getHealthColor('online') }}>
              {data.metrics.cache_hit_rate.toFixed(1)}%
            </div>
            <div style={{ fontSize: '12px', color: isDarkMode ? '#9ca3af' : '#6b7280' }}>
              Cache Hit Rate
            </div>
          </div>
        </div>
      </div>

      {/* Rotas Trending com Alertas de Crise */}
      <div style={cardStyle}>
        <h2 style={{ fontSize: '20px', fontWeight: '700', margin: '0 0 16px 0' }}>
          Rotas em Alta
          {data.trending_routes.some(r => r.crisis_level === 'critical') && (
            <span style={{ marginLeft: '12px', color: '#ef4444', fontSize: '14px', fontWeight: '600' }}>
              ALERTA: Crise Iminente Detectada!
            </span>
          )}
        </h2>
        
        <div style={{ maxHeight: '400px', overflowY: 'auto' }}>
          {data.trending_routes.map((route, index) => (
            <div
              key={index}
              className="crisis-row"
              style={{
                borderLeftColor: getCrisisColor(route.crisis_level),
                backgroundColor: route.crisis_level === 'critical' ? (isDarkMode ? 'rgba(239, 68, 68, 0.1)' : 'rgba(239, 68, 68, 0.05)') : 'transparent'
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                  <div style={{ fontSize: '16px', fontWeight: '600' }}>
                    {route.origin} -> {route.destination}
                    {route.crisis_level && route.crisis_level !== 'normal' && (
                      <span style={crisisBadgeStyle(route.crisis_level)}>
                        {route.crisis_level === 'critical' ? 'CRISE IMINENTE' : 'ALERTA'}
                      </span>
                    )}
                  </div>
                  <div style={{ fontSize: '12px', color: isDarkMode ? '#9ca3af' : '#6b7280' }}>
                    {route.count} acessos
                    {route.report_count !== undefined && (
                      <span style={{ marginLeft: '8px' }}>
                        {route.report_count} denúncias
                      </span>
                    )}
                  </div>
                </div>
                
                <div style={{ textAlign: 'right' }}>
                  <div style={{ fontSize: '14px', fontWeight: '600' }}>
                    Score: {route.crisis_score?.toFixed(1) || '0.0'}
                  </div>
                  <div style={{ fontSize: '12px', color: isDarkMode ? '#9ca3af' : '#6b7280' }}>
                    {new Date(route.last_search).toLocaleString('pt-BR')}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Denúncias Recentes */}
      <div style={cardStyle}>
        <h2 style={{ fontSize: '20px', fontWeight: '700', margin: '0 0 16px 0' }}>
          Denúncias Recentes
        </h2>
        
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '16px' }}>
          {data.recent_reports.map((report, index) => (
            <div key={index} style={{ textAlign: 'center' }}>
              <div style={{ fontSize: '20px', fontWeight: '700', color: getCrisisColor(report.severity) }}>
                {report.count}
              </div>
              <div style={{ fontSize: '14px', fontWeight: '600' }}>
                {report.tipo_problema}
              </div>
              <div style={{ fontSize: '12px', color: isDarkMode ? '#9ca3af' : '#6b7280' }}>
                {report.severity === 'high' ? 'Alta' : report.severity === 'medium' ? 'Média' : 'Baixa'} severidade
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default AdminDashboard;
