import React, { useEffect } from 'react';
import { UserProfile } from '../types/api';

interface UserCardProps {
  user: UserProfile;
  onToggleSunMode?: () => void;
  isSunMode?: boolean;
  className?: string;
}

const UserCard: React.FC<UserCardProps> = ({ 
  user,
  onToggleSunMode,
  isSunMode = false,
  className = '' 
}) => {
  useEffect(() => {
    if (isSunMode) {
      document.body.classList.add('high-contrast');
    } else {
      document.body.classList.remove('high-contrast');
    }
  }, [isSunMode]);

  return (
    <div className={`flex items-center gap-3 ${className}`}>
      <button
        className={`w-10 h-10 rounded-full flex items-center justify-center transition-all active:scale-95 ${
          isSunMode 
            ? 'bg-yellow-500/20 text-yellow-400 hover:bg-yellow-500/30' 
            : 'bg-tertiary/10 text-tertiary hover:bg-tertiary/20'
        }`}
        onClick={onToggleSunMode}
        title={isSunMode ? 'Desativar Modo Sol' : 'Modo Sol (Alto Contraste)'}
      >
        <span className="material-symbols-outlined" style={{ fontVariationSettings: '"FILL" 1' }}>
          {isSunMode ? 'dark_mode' : 'light_mode'}
        </span>
      </button>
      <div className="h-8 w-8 rounded-full bg-surface-container-highest overflow-hidden border border-outline-variant/30">
        <img
          className="w-full h-full object-cover"
          alt={user.name}
          src={user.avatar}
        />
      </div>
    </div>
  );
};

export default UserCard;
