import { useEffect, useState, useCallback, useRef } from 'react';
import { io, Socket } from 'socket.io-client';

interface BusUpdate {
  device_id: string;
  lat: number;
  lng: number;
  speed: number;
  timestamp: string;
}

interface BIAlert {
  alert: string;
  avg_speed: number;
  baseline: number;
  reduction: number;
  bus_count: number;
  timestamp: string;
}

interface GeofenceAlert {
  device_id: string;
  lat: number;
  lng: number;
  alert: string;
  fence: string;
  timestamp: string;
}

export interface RealtimeData {
  buses: Map<string, BusUpdate>;
  biAlert: BIAlert | null;
  geofenceAlert: GeofenceAlert | null;
}

export function useRealtime() {
  const [data, setData] = useState<RealtimeData>({
    buses: new Map(),
    biAlert: null,
    geofenceAlert: null,
  });
  const [isConnected, setIsConnected] = useState(false);
  const socketRef = useRef<Socket | null>(null);

  const connect = useCallback(() => {
    const wsUrl = import.meta.env.VITE_WS_URL || 'ws://localhost:8081/ws';
    
    // Usar WebSocket nativo em vez de Socket.IO (backend usa gorilla/websocket)
    const ws = new WebSocket(wsUrl.replace('ws://', 'ws://').replace('http://', 'ws://'));
    
    ws.onopen = () => {
      console.log('[WebSocket] Conectado ao servidor');
      setIsConnected(true);
    };

    ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        
        // Verificar tipo de mensagem
        if (message.device_id && message.lat && message.lng) {
          // Atualização de ônibus
          setData((prev) => {
            const newBuses = new Map(prev.buses);
            newBuses.set(message.device_id, message);
            return { ...prev, buses: newBuses };
          });
        } else if (message.alert === 'CONGESTION_DETECTED') {
          // Alerta de BI
          setData((prev) => ({ ...prev, biAlert: message }));
        } else if (message.alert === 'GEOFENCE_BREACH') {
          // Alerta de Geofencing
          setData((prev) => ({ ...prev, geofenceAlert: message }));
        }
      } catch (error) {
        console.error('[WebSocket] Erro ao parsear mensagem:', error);
      }
    };

    ws.onerror = (error) => {
      console.error('[WebSocket] Erro:', error);
      setIsConnected(false);
    };

    ws.onclose = () => {
      console.log('[WebSocket] Desconectado');
      setIsConnected(false);
      // Tentar reconectar após 5 segundos
      setTimeout(connect, 5000);
    };

    socketRef.current = ws as any;

    return () => {
      ws.close();
    };
  }, []);

  useEffect(() => {
    const cleanup = connect();
    return () => {
      if (socketRef.current) {
        socketRef.current.close();
      }
      if (cleanup) {
        cleanup();
      }
    };
  }, [connect]);

  const clearBIAlert = useCallback(() => {
    setData((prev) => ({ ...prev, biAlert: null }));
  }, []);

  const clearGeofenceAlert = useCallback(() => {
    setData((prev) => ({ ...prev, geofenceAlert: null }));
  }, []);

  return {
    data,
    isConnected,
    clearBIAlert,
    clearGeofenceAlert,
  };
}
