import React, { useEffect, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { BusLocation } from '../types/api';

interface MapContainerProps {
  busLocations?: BusLocation[];
  className?: string;
}

const MapContainer: React.FC<MapContainerProps> = ({
  busLocations = [],
  className = ''
}) => {
  // Injetar ônibus fake TESTE-99 no estado inicial para teste
  const [liveBuses, setLiveBuses] = useState<BusLocation[]>([
    {
      id: 'TESTE-99',
      lat: 50,
      lng: 50,
      speed: 35,
      heading: 90,
      isRecent: true,
    }
  ]);

  useEffect(() => {
    console.log('--- COMPONENTE MAPA MONTADO ---');
    // WebSocket connection for real-time updates
    const wsUrl = import.meta.env.VITE_WS_URL || 'ws://localhost:8080';
    const ws = new WebSocket(`${wsUrl}/api/v1/telemetry/ws`);

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);

      setLiveBuses((prev) => {
        const existing = prev.find((b) => b.id === data.id);
        if (existing) {
          return prev.map((b) =>
            b.id === data.id
              ? { ...b, lat: data.lat, lng: data.lng, speed: data.speed, isRecent: true }
              : b
          );
        }
        return [...prev, {
          id: data.id,
          lat: data.lat,
          lng: data.lng,
          speed: data.speed,
          heading: data.heading || 0,
          isRecent: true,
        }];
      });
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    return () => ws.close();
  }, []);

  return (
    <div className={`h-40 w-full relative ${className}`} style={{ position: 'relative' }}>
      {/* Fundo sólido temporário substituindo Google Maps */}
      <div className="w-full h-full bg-blue-900 flex items-center justify-center">
        <span className="text-white font-bold text-2xl">MAPA ATIVO</span>
      </div>
      <div className="absolute inset-0 bg-gradient-to-t from-surface-container-low to-transparent" />
      
      <AnimatePresence>
        {liveBuses?.map((bus) => (
          <motion.div
            key={bus.id}
            initial={{ opacity: 0, scale: 0 }}
            animate={{
              opacity: bus.isRecent ? 1 : 0.3,
              scale: 1,
              top: `${bus.lat}%`,
              left: `${bus.lng}%`,
            }}
            transition={{
              type: 'spring',
              stiffness: 300,
              damping: 30,
            }}
            style={{ position: 'absolute' }}
            className={`w-6 h-6 bg-secondary rounded-full flex items-center justify-center border-2 border-surface shadow-xl ${
              bus.isRecent ? 'glow-green z-20' : 'bg-secondary/30 border-surface/20 z-10 backdrop-blur-sm'
            }`}
          >
            <span className="material-symbols-outlined text-[12px] text-on-secondary font-bold">
              directions_bus
            </span>
          </motion.div>
        ))}
      </AnimatePresence>
      
      <div className="absolute bottom-4 left-6 flex items-center gap-2">
        <span className="text-on-surface font-headline font-bold text-sm">
          Ver Trajeto Completo
        </span>
        <span className="material-symbols-outlined text-sm">arrow_forward</span>
      </div>
    </div>
  );
};

export default MapContainer;
