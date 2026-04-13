// Mock API Service for TranspRota Frontend
// This simulates backend responses for development purposes

export interface FleetStatus {
  total_active_buses: number;
  total_geofence_alerts: number;
  average_speed: number;
  total_buses: number;
  offline_buses: number;
  last_updated: string;
}

export interface ComplianceScore {
  score: number;
  trend: 'up' | 'down' | 'stable';
  factors: {
    geofence: number;
    speed: number;
    offline: number;
  };
}

export interface AuditLog {
  id: string;
  device_id: string;
  action: string;
  timestamp: string;
  user: string;
}

// Telemetry Engine - Simula dados dinâmicos
class TelemetryEngine {
  private listeners: Set<() => void> = new Set();
  private interval: NodeJS.Timeout | null = null;

  // Estado atual
  public fleetStatus: FleetStatus = {
    total_active_buses: 42,
    total_geofence_alerts: 3,
    average_speed: 45.2,
    total_buses: 50,
    offline_buses: 8,
    last_updated: new Date().toISOString(),
  };

  public complianceScore: ComplianceScore = {
    score: 98,
    trend: 'stable',
    factors: {
      geofence: 95,
      speed: 100,
      offline: 99,
    },
  };

  public auditLogs: AuditLog[] = [
    {
      id: '1',
      device_id: 'bus-001',
      action: 'Rota alterada manualmente',
      timestamp: new Date(Date.now() - 86400000).toISOString(),
      user: 'admin',
    },
    {
      id: '2',
      device_id: 'bus-005',
      action: 'Alerta de geofencing criado',
      timestamp: new Date(Date.now() - 172800000).toISOString(),
      user: 'supervisor',
    },
    {
      id: '3',
      device_id: 'bus-012',
      action: 'Velocidade excessiva detectada',
      timestamp: new Date(Date.now() - 259200000).toISOString(),
      user: 'system',
    },
  ];

  // Variar dados aleatoriamente
  private varyData() {
    // Variação de frota (40-45 ativos)
    this.fleetStatus.total_active_buses = Math.floor(40 + Math.random() * 5);
    this.fleetStatus.offline_buses = this.fleetStatus.total_buses - this.fleetStatus.total_active_buses;
    this.fleetStatus.average_speed = parseFloat((40 + Math.random() * 15).toFixed(1));
    this.fleetStatus.total_geofence_alerts = Math.floor(Math.random() * 5);
    this.fleetStatus.last_updated = new Date().toISOString();

    // Cálculo de compliance (baseado nos fatores)
    const geofenceFactor = 90 + Math.random() * 10;
    const speedFactor = 95 + Math.random() * 5;
    const offlineFactor = 92 + Math.random() * 8;
    
    this.complianceScore.factors.geofence = parseFloat(geofenceFactor.toFixed(0));
    this.complianceScore.factors.speed = parseFloat(speedFactor.toFixed(0));
    this.complianceScore.factors.offline = parseFloat(offlineFactor.toFixed(0));
    
    this.complianceScore.score = Math.floor((geofenceFactor + speedFactor + offlineFactor) / 3);
    
    // Trend
    const prevScore = this.complianceScore.score;
    if (Math.random() > 0.7) {
      this.complianceScore.score = Math.max(85, Math.min(100, this.complianceScore.score + (Math.random() > 0.5 ? 1 : -1)));
    }
    this.complianceScore.trend = this.complianceScore.score > prevScore ? 'up' : this.complianceScore.score < prevScore ? 'down' : 'stable';

    // Adicionar novo log de auditoria aleatoriamente
    if (Math.random() > 0.8) {
      const actions = ['Velocidade excessiva', 'Fora de rota', 'Parada não autorizada', 'Conexão perdida'];
      const devices = ['bus-001', 'bus-005', 'bus-012', 'bus-023', 'bus-034'];
      const users = ['system', 'admin', 'supervisor'];
      
      const newLog: AuditLog = {
        id: Date.now().toString(),
        device_id: devices[Math.floor(Math.random() * devices.length)],
        action: actions[Math.floor(Math.random() * actions.length)],
        timestamp: new Date().toISOString(),
        user: users[Math.floor(Math.random() * users.length)],
      };
      
      this.auditLogs.unshift(newLog);
      if (this.auditLogs.length > 10) {
        this.auditLogs.pop();
      }
    }

    // Notificar listeners
    this.listeners.forEach(listener => listener());
  }

  // Iniciar engine
  start(intervalMs: number = 5000) {
    if (this.interval) return;
    this.interval = setInterval(() => this.varyData(), intervalMs);
  }

  // Parar engine
  stop() {
    if (this.interval) {
      clearInterval(this.interval);
      this.interval = null;
    }
  }

  // Adicionar listener
  subscribe(listener: () => void) {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  // Obter dados atuais
  getFleetStatus(): FleetStatus {
    return { ...this.fleetStatus };
  }

  getComplianceScore(): ComplianceScore {
    return { ...this.complianceScore };
  }

  getAuditLogs(): AuditLog[] {
    return [...this.auditLogs];
  }
}

// Instância única do engine
const telemetryEngine = new TelemetryEngine();

// API functions
export const api = {
  // Iniciar engine de telemetria
  startTelemetry: (intervalMs: number = 5000) => {
    telemetryEngine.start(intervalMs);
  },

  // Parar engine de telemetria
  stopTelemetry: () => {
    telemetryEngine.stop();
  },

  // Subscribe para atualizações
  subscribeTelemetry: (callback: () => void) => {
    return telemetryEngine.subscribe(callback);
  },

  // Get fleet status (monitorado via Go)
  getFleetStatus: (): FleetStatus => {
    return telemetryEngine.getFleetStatus();
  },

  // Get compliance score (cálculo matemático)
  getComplianceScore: (): ComplianceScore => {
    return telemetryEngine.getComplianceScore();
  },

  // Get audit logs (últimos eventos)
  getAuditLogs: (): AuditLog[] => {
    return telemetryEngine.getAuditLogs();
  },
};

export default api;
