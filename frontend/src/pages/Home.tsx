import React, { useState } from 'react';
import BottomSheet from '../components/BottomSheet';
import UserCard from '../components/UserCard';
import RouteList from '../components/RouteList';
import ConnectionStatus from '../components/ConnectionStatus';
import { useFleetStatus } from '../hooks/useFleetStatus';
import { Route, Stop, UserProfile } from '../types/api';

const Home: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'explore' | 'plan' | 'routes' | 'profile'>('routes');
  const [isSunMode, setIsSunMode] = useState(false);
  
  const { data: fleetData } = useFleetStatus();

  // Mock data - will be replaced with API calls
  const route: Route = {
    id: '006',
    number: '006',
    name: 'Eixo Anhanguera',
    direction: 'Terminal Novo Mundo',
    status: 'active',
    eta: 8,
    trafficStatus: 'fluido',
  };

  const stops: Stop[] = [
    {
      id: '1',
      name: 'Terminal Padre Pelágio',
      status: 'past',
    },
    {
      id: '2',
      name: 'Terminal Praça da Bíblia',
      status: 'current',
    },
    {
      id: '3',
      name: 'Praça A',
      status: 'future',
      eta: 8,
    },
    {
      id: '4',
      name: 'Terminal Novo Mundo',
      status: 'end',
      estimatedArrival: '18:45',
    },
  ];

  const user: UserProfile = {
    name: 'Usuário',
    avatar: 'https://lh3.googleusercontent.com/aida-public/AB6AXuAPrWuPK_FEk-kI4Ts0AXoEXlAP9epB5y_NJoQwyh5EL2B36E1TEEihnr2bJA3_8a0tunIb7MF1mVasygfoCVKc_MpbBGI1zz3sYj7S-HGLZ-Oy3LHUgZgYU3aBFod7TrqHo1Y-Fcst9_AA38t9Vo4y6HYUmzIhBQvNasqDk2KJCHLYAxkjZIV4uHU8CIKCpS03gIdjPRGBjPtY_dKydTYnIJMrXO9tPO7KwEWjTWm1yBfYW9a5thF_fLCgQk2bMW_weBawjwGWboU',
    level: 'Explorador Urbano',
    xp: 750,
    xpToNextLevel: 1600,
    streak: 12,
  };

  const handleToggleSunMode = () => {
    setIsSunMode(!isSunMode);
    document.body.classList.toggle('sun-mode');
  };

  return (
    <div className={`min-h-screen ${isSunMode ? 'sun-mode' : ''}`}>
      {/* TopAppBar Shell */}
      <header className="fixed top-0 w-full z-50 bg-[#0b1326]/60 backdrop-blur-xl flex items-center justify-between px-6 h-16 w-full shadow-[0_0_24px_rgba(173,198,255,0.06)]">
        <div className="flex items-center gap-4">
          <button className="hover:bg-[#adc6ff]/10 transition-colors p-2 rounded-full active:scale-90 transition-transform">
            <span className="material-symbols-outlined text-[#adc6ff]">menu</span>
          </button>
          <h1 className="font-['Manrope'] font-bold text-lg tracking-tight text-[#adc6ff]">
            Transprota
          </h1>
        </div>
        <div className="flex items-center gap-4">
          <ConnectionStatus />
          <UserCard user={user} onToggleSunMode={handleToggleSunMode} isSunMode={isSunMode} />
        </div>
      </header>

      <main className="pt-24 px-6 max-w-2xl mx-auto">
        {/* Section: Title */}
        <div className="mb-8 ml-2">
          <h2 className="font-headline font-extrabold text-3xl tracking-tight text-on-surface">
            Analisar Rotas
          </h2>
          <p className="text-on-surface-variant text-sm mt-1">
            Sincronize seu trajeto com o pulso da cidade.
          </p>
        </div>

        {/* Search Bar */}
        <div className="relative group mb-10">
          <div className="absolute inset-y-0 left-4 flex items-center pointer-events-none">
            <span className="material-symbols-outlined text-outline">search</span>
          </div>
          <input
            className="w-full h-14 pl-12 pr-4 bg-surface-variant/60 backdrop-blur-[30px] border-none rounded-xl text-on-surface placeholder-on-surface-variant focus:ring-1 focus:ring-secondary/50 transition-all outline-none"
            placeholder="Pesquisar número ou nome da linha"
            type="text"
          />
        </div>

        {/* Recent/Suggested Horizontal Bento */}
        <section className="mb-12">
          <div className="flex justify-between items-end mb-4 px-2">
            <span className="font-label text-[10px] font-bold uppercase tracking-[0.1em] text-outline">
              Sugestões de Linha
            </span>
            <span className="text-xs text-primary font-medium">Ver todas</span>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="bg-surface-container-high p-4 rounded-xl flex items-center gap-4 hover:bg-surface-container-highest transition-colors cursor-pointer group">
              <div className="w-10 h-10 rounded-lg bg-secondary/10 flex items-center justify-center glow-green">
                <span className="material-symbols-outlined text-on-secondary font-bold">
                  directions_bus
                </span>
              </div>
              <div>
                <div className="font-headline font-bold text-lg leading-tight group-hover:text-secondary transition-colors">
                  006
                </div>
                <div className="text-[10px] text-on-surface-variant font-medium uppercase tracking-wider">
                  Eixo Anhanguera
                </div>
              </div>
            </div>
            {/* Skeleton Loading Example */}
            <div className="bg-surface-container-high p-4 rounded-xl flex items-center gap-4">
              <div className="w-10 h-10 rounded-lg skeleton opacity-30" />
              <div className="flex-1 space-y-2">
                <div className="h-4 w-1/2 skeleton rounded opacity-30" />
                <div className="h-2 w-3/4 skeleton rounded opacity-30" />
              </div>
            </div>
          </div>
        </section>

        {/* Main Content: Detailed Route View */}
        <RouteList 
          route={route} 
          stops={stops} 
          confidence={fleetData?.positions[0]?.confidence || 75}
        />
      </main>

      {/* BottomNavBar Shell */}
      <BottomSheet activeTab={activeTab} onTabChange={setActiveTab} />
    </div>
  );
};

export default Home;
