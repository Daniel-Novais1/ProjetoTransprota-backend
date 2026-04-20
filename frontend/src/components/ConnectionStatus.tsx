import React, { useEffect, useState } from 'react';

const ConnectionStatus: React.FC = () => {
  const [status, setStatus] = useState<'connecting' | 'connected' | 'error'>('connecting');

  useEffect(() => {
    const checkHealth = async () => {
      try {
        const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080';
        const url = `${apiUrl}/api/v1/health`;
        console.log('🔍 [HEALTH] Checking:', url);
        const response = await fetch(url);
        console.log('🔍 [HEALTH] Status:', response.status, response.statusText);
        const data = await response.json();
        console.log('🔍 [HEALTH] Response:', data);
        if (response.ok) {
          setStatus('connected');
          alert(`✅ Health Check OK! Status: ${response.status}`);
        } else {
          setStatus('error');
          alert(`❌ Health Check Failed! Status: ${response.status}`);
        }
      } catch (error) {
        console.error('🔍 [HEALTH] Failed:', error);
        setStatus('error');
        alert(`❌ Health Check Error: ${error}`);
      }
    };

    checkHealth();
    const interval = setInterval(checkHealth, 30000); // Check every 30 seconds

    return () => clearInterval(interval);
  }, []);

  return (
    <div className="flex items-center gap-2">
      <div
        className={`w-2 h-2 rounded-full ${
          status === 'connected' ? 'bg-green-500 glow-green' : 
          status === 'connecting' ? 'bg-yellow-500 animate-pulse' : 
          'bg-red-500'
        }`}
      />
      <span className="text-xs text-on-surface-variant">
        {status === 'connected' ? 'Conectado' : 
         status === 'connecting' ? 'Conectando...' : 
         'Erro'}
      </span>
    </div>
  );
};

export default ConnectionStatus;
