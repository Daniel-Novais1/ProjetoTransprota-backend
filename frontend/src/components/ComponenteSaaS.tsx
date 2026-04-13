import React, { useState, useEffect } from 'react';
import { api, FleetStatus, ComplianceScore, AuditLog } from '@/services/api';

export const ComponenteSaaS = () => {
  const [fleetStatus, setFleetStatus] = useState<FleetStatus>(api.getFleetStatus());
  const [complianceScore, setComplianceScore] = useState<ComplianceScore>(api.getComplianceScore());
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>(api.getAuditLogs());

  useEffect(() => {
    // Iniciar engine de telemetria
    api.startTelemetry(5000);

    // Subscribe para atualizações
    const unsubscribe = api.subscribeTelemetry(() => {
      setFleetStatus(api.getFleetStatus());
      setComplianceScore(api.getComplianceScore());
      setAuditLogs(api.getAuditLogs());
    });

    // Cleanup
    return () => {
      unsubscribe();
      api.stopTelemetry();
    };
  }, []);

  const getTrendIcon = (trend: string) => {
    switch (trend) {
      case 'up': return '📈';
      case 'down': return '📉';
      default: return '➡️';
    }
  };

  const getTrendColor = (trend: string) => {
    switch (trend) {
      case 'up': return 'text-green-400';
      case 'down': return 'text-red-400';
      default: return 'text-slate-400';
    }
  };
  return (
    <div className="flex min-h-screen bg-slate-950">
      {/* Título Centralizado */}
      <div className="absolute inset-0 flex items-center justify-center z-50">
        <h1 className="text-5xl font-bold text-white bg-slate-900 px-8 py-4 rounded-lg shadow-2xl border-2 border-green-500">
          ESTOU NO DASHBOARD B2B - SEM MAPA
        </h1>
      </div>

      {/* Sidebar */}
      <aside className="w-64 bg-slate-900 shadow-lg">
        <div className="p-6">
          <h1 className="text-xl font-bold text-white">TranspRota</h1>
          <p className="text-sm text-slate-400 mt-1">Dashboard de Compliance</p>
        </div>
        <nav className="mt-6">
          <a href="#/" className="flex items-center px-6 py-3 text-slate-300 hover:bg-slate-800">
            <span className="mr-3">🏠</span>
            Home
          </a>
          <a href="#/dashboard" className="flex items-center px-6 py-3 text-slate-300 bg-slate-800">
            <span className="mr-3">📊</span>
            Dashboard
          </a>
          <a href="#/settings" className="flex items-center px-6 py-3 text-slate-300 hover:bg-slate-800">
            <span className="mr-3">⚙️</span>
            Configurações
          </a>
        </nav>
      </aside>

      {/* Main Content */}
      <main className="flex-1 p-8">
        <h1 className="text-2xl font-bold text-white mb-6">Dashboard de Compliance</h1>
        
        {/* Grid de 3 Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          {/* Card 1: Status de Frota (Monitorado via Go) */}
          <div className="bg-slate-900 rounded-lg shadow-lg p-6 border-l-4 border-green-500">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-slate-400">Status de Frota</p>
                <p className="text-3xl font-bold text-white mt-2">{fleetStatus.total_active_buses}/{fleetStatus.total_buses}</p>
                <p className="text-xs text-slate-500 mt-1">Ônibus ativos</p>
                <p className="text-xs text-slate-500 mt-1">Velocidade média: {fleetStatus.average_speed} km/h</p>
              </div>
              <div className="bg-green-900 p-3 rounded-full">
                <span className="text-2xl">🚌</span>
              </div>
            </div>
          </div>

          {/* Card 2: Score de Compliance (Cálculo matemático) */}
          <div className="bg-slate-900 rounded-lg shadow-lg p-6 border-l-4 border-blue-500">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-slate-400">Score de Compliance</p>
                <p className="text-3xl font-bold text-white mt-2">{complianceScore.score}%</p>
                <p className={`text-xs mt-1 ${getTrendColor(complianceScore.trend)}`}>
                  {getTrendIcon(complianceScore.trend)} {complianceScore.trend.toUpperCase()}
                </p>
                <p className="text-xs text-slate-500 mt-1">Geofence: {complianceScore.factors.geofence}%</p>
              </div>
              <div className="bg-blue-900 p-3 rounded-full">
                <span className="text-2xl">📊</span>
              </div>
            </div>
          </div>

          {/* Card 3: Logs de Auditoria (Últimos eventos) */}
          <div className="bg-slate-900 rounded-lg shadow-lg p-6 border-l-4 border-purple-500">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-slate-400">Logs de Auditoria</p>
                <p className="text-3xl font-bold text-white mt-2">{auditLogs.length}</p>
                <p className="text-xs text-slate-500 mt-1">eventos recentes</p>
                <p className="text-xs text-slate-500 mt-1">Último: {new Date(auditLogs[0]?.timestamp).toLocaleTimeString()}</p>
              </div>
              <div className="bg-purple-900 p-3 rounded-full">
                <span className="text-2xl">📋</span>
              </div>
            </div>
          </div>
        </div>

        {/* Lista de Logs de Auditoria */}
        <div className="bg-slate-900 rounded-lg shadow-lg p-6">
          <h2 className="text-xl font-bold text-white mb-4">Últimos Eventos de Auditoria</h2>
          <div className="space-y-3">
            {auditLogs.slice(0, 5).map((log) => (
              <div key={log.id} className="flex items-center justify-between bg-slate-800 p-3 rounded-lg">
                <div className="flex items-center gap-3">
                  <span className="text-2xl">📝</span>
                  <div>
                    <p className="text-sm text-white font-medium">{log.action}</p>
                    <p className="text-xs text-slate-400">{log.device_id} • {log.user}</p>
                  </div>
                </div>
                <p className="text-xs text-slate-500">{new Date(log.timestamp).toLocaleString()}</p>
              </div>
            ))}
          </div>
        </div>

        <p className="text-slate-400">Espaço reservado para lógica de auditoria e telemetria</p>
      </main>
    </div>
  );
};
