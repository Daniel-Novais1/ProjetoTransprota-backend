import { useQuery } from '@tanstack/react-query';
import { LatestPositionsResponse } from '../types/api';

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export const useFleetStatus = () => {
  return useQuery({
    queryKey: ['fleetStatus'],
    queryFn: async (): Promise<LatestPositionsResponse> => {
      const response = await fetch(`${API_BASE}/api/v1/telemetry/latest`);
      if (!response.ok) {
        throw new Error('Failed to fetch fleet status');
      }
      const data = await response.json();
      
      // Log se array vier vazio
      if (data.p && Array.isArray(data.p) && data.p.length === 0) {
        console.log('Busca realizada, mas o array veio com tamanho 0');
      }
      
      return data;
    },
    staleTime: 5000, // 5 seconds as requested
    refetchInterval: 5000, // Auto-refresh every 5 seconds
  });
};
