import React from 'react';
import { Route, Stop } from '../types/api';

interface RouteListProps {
  route: Route;
  stops: Stop[];
  confidence?: number;
  className?: string;
}

const RouteList: React.FC<RouteListProps> = ({ route, stops, confidence = 75, className = '' }) => {
  
  const getConfidenceColor = (conf: number) => {
    if (conf > 80) return 'text-green-400 shadow-green-500/50 drop-shadow-lg';
    if (conf < 50) return 'text-yellow-400 shadow-yellow-500/50 drop-shadow-lg';
    return 'text-on-surface';
  };
  return (
    <div className={`space-y-6 ${className}`}>
      {/* Route Header Card */}
      <div className="bg-surface-container-low rounded-2xl overflow-hidden shadow-2xl">
        <div className="p-6 flex justify-between items-start">
          <div>
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-secondary-container/20 text-secondary text-[11px] font-bold tracking-widest uppercase mb-3">
              <span className="w-1.5 h-1.5 rounded-full bg-secondary kinetic-pulse" />
              Em Operação
            </div>
            <h3 className="font-headline font-extrabold text-2xl text-on-surface">
              {route.number} - {route.name}
            </h3>
            <p className="text-on-surface-variant text-sm mt-1">
              Sentido: {route.direction}
            </p>
          </div>
          <button
            className="w-12 h-12 rounded-full bg-primary-container flex items-center justify-center shadow-lg active:scale-95 transition-transform overflow-hidden relative group"
            onClick={() => alert('Alerta de proximidade ativado! ✨')}
          >
            <span className="material-symbols-outlined text-on-primary-container" style={{ fontVariationSettings: '"FILL" 1' }}>
              notifications
            </span>
          </button>
        </div>

        {/* Timeline / Stops */}
        <div className="p-6 pt-2">
          <div className="space-y-0">
            {stops.map((stop, index) => (
              <div key={stop.id} className="flex gap-6 items-start group">
                <div className="flex flex-col items-center h-full">
                  {stop.status === 'current' ? (
                    <div className="w-5 h-5 rounded-full bg-secondary flex items-center justify-center ring-4 ring-secondary/20 kinetic-pulse z-10 glow-green">
                      <span className="material-symbols-outlined text-[12px] text-on-secondary font-black">
                        check
                      </span>
                    </div>
                  ) : (
                    <div
                      className={`w-3 h-3 rounded-full ${
                        stop.status === 'past'
                          ? 'bg-outline-variant'
                          : 'border-2 border-outline-variant bg-surface group-hover:border-primary transition-colors'
                      } mt-1.5`}
                    />
                  )}
                  {index < stops.length - 1 && (
                    <div className={`w-0.5 ${
                      stop.status === 'current' 
                        ? 'h-20 bg-gradient-to-b from-secondary to-outline-variant/30'
                        : 'h-16 bg-outline-variant/30'
                    }`} />
                  )}
                </div>
                <div className={`${stop.status !== 'end' ? 'pb-8' : ''}`}>
                  <h4
                    className={`${
                      stop.status === 'past'
                        ? 'text-on-surface-variant font-medium line-through'
                        : 'on-surface font-medium group-hover:text-primary transition-colors'
                    } ${stop.status === 'current' ? 'font-bold text-lg' : ''}`}
                  >
                    {stop.name}
                  </h4>
                  {stop.status === 'past' && (
                    <span className="text-[10px] text-outline font-bold tracking-widest uppercase">
                      Passou há 12 min
                    </span>
                  )}
                  {stop.status === 'current' && (
                    <div className="flex items-center gap-2 mt-1">
                      <span className="px-2 py-0.5 rounded bg-secondary/10 text-secondary text-[10px] font-bold">
                        VOCÊ ESTÁ AQUI
                      </span>
                      <span className="text-on-surface-variant text-xs">
                        • Embarque Imediato
                      </span>
                    </div>
                  )}
                  {stop.status === 'future' && stop.eta !== undefined && (
                    <div className="flex items-center gap-3 mt-1">
                      <span className={`text-[2.5rem] font-headline font-extrabold leading-none tracking-tighter ${getConfidenceColor(confidence)}`}>
                        {String(stop.eta).padStart(2, '0')}
                      </span>
                      <span className="text-[10px] text-outline font-bold tracking-widest uppercase leading-tight">
                        MINUTOS<br />RESTANTES
                      </span>
                    </div>
                  )}
                  {stop.status === 'end' && stop.estimatedArrival && (
                    <span className="text-[10px] text-outline font-bold tracking-widest uppercase">
                      Chegada est. {stop.estimatedArrival}
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Gamification / Streak Card */}
      <div className="grid grid-cols-5 gap-4">
        <div className="col-span-3 bg-surface-container-high p-5 rounded-2xl border border-outline-variant/10">
          <div className="flex items-start justify-between">
            <div>
              <span className="text-[10px] text-tertiary font-bold tracking-[0.15em] uppercase">
                Status de Passageiro
              </span>
              <h4 className="font-headline font-bold text-xl mt-1 text-on-surface">
                {user.level}
              </h4>
            </div>
            <span className="material-symbols-outlined text-tertiary" style={{ fontVariationSettings: '"FILL" 1' }}>
              military_tech
            </span>
          </div>
          <div className="mt-4 bg-surface-variant h-1.5 rounded-full overflow-hidden">
            <div
              className="bg-tertiary h-full rounded-full shadow-[0_0_8px_#ffb786]"
              style={{ width: `${(user.xp / user.xpToNextLevel) * 100}%` }}
            />
          </div>
          <p className="text-[10px] text-on-surface-variant mt-2 font-medium">
            {user.xpToNextLevel - user.xp} XP para o nível VIP
          </p>
        </div>
        <div className="col-span-2 bg-[#ffb786]/10 p-5 rounded-2xl flex flex-col justify-between border border-tertiary/10">
          <span className="material-symbols-outlined text-tertiary text-3xl" style={{ fontVariationSettings: 'FILL 1' }}>
            local_fire_department
          </span>
          <div>
            <div className="text-2xl font-black text-on-surface font-headline leading-none">
              {user.streak}
            </div>
            <div className="text-[9px] text-tertiary font-bold uppercase tracking-wider mt-1">
              Dias de sequência
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

// Mock user data for the component
const user = {
  level: 'Explorador Urbano',
  xp: 750,
  xpToNextLevel: 1600,
  streak: 12,
};

export default RouteList;
