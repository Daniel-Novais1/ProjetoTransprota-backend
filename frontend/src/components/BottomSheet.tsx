import React from 'react';

interface BottomSheetProps {
  activeTab?: 'explore' | 'plan' | 'routes' | 'profile';
  onTabChange?: (tab: 'explore' | 'plan' | 'routes' | 'profile') => void;
  className?: string;
}

const BottomSheet: React.FC<BottomSheetProps> = ({ 
  activeTab = 'routes',
  onTabChange,
  className = '' 
}) => {
  const tabs = [
    { id: 'explore' as const, icon: 'explore', label: 'Explorar' },
    { id: 'plan' as const, icon: 'route', label: 'Planejar' },
    { id: 'routes' as const, icon: 'directions_bus', label: 'Rotas' },
    { id: 'profile' as const, icon: 'person', label: 'Perfil' },
  ];

  return (
    <nav className={`fixed bottom-0 left-0 w-full flex justify-around items-center px-4 pt-2 pb-6 bg-[#0b1326]/80 backdrop-blur-2xl z-50 rounded-t-[1.5rem] shadow-[0_-4px_20px_rgba(0,0,0,0.4)] ${className}`}>
      {tabs.map((tab) => (
        <a
          key={tab.id}
          className={`flex flex-col items-center justify-center ${
            activeTab === tab.id
              ? 'text-[#adc6ff] bg-[#adc6ff]/10 rounded-xl px-4 py-2'
              : 'text-slate-500 hover:text-slate-300'
          } scale-95 active:scale-90 transition-all duration-200`}
          href="#"
          onClick={(e) => {
            e.preventDefault();
            onTabChange?.(tab.id);
          }}
        >
          <span className="material-symbols-outlined">{tab.icon}</span>
          <span className="font-['Inter'] text-[11px] font-medium uppercase tracking-[0.05em] mt-1">
            {tab.label}
          </span>
        </a>
      ))}
    </nav>
  );
};

export default BottomSheet;
