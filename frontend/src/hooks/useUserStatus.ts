import { useQuery } from '@tanstack/react-query';

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export interface UserStatusResponse {
  userId: string;
  username: string;
  level: string;
  xp: number;
  xpToNextLevel: number;
  streak: number;
  lastCheckIn?: string;
}

export const useUserStatus = (userId?: string) => {
  return useQuery({
    queryKey: ['userStatus', userId],
    queryFn: async (): Promise<UserStatusResponse> => {
      if (!userId) {
        throw new Error('User ID is required');
      }
      const response = await fetch(`${API_BASE}/api/v1/user/status?userId=${userId}`);
      if (!response.ok) {
        throw new Error('Failed to fetch user status');
      }
      return response.json();
    },
    staleTime: 5000, // 5 seconds as requested
    enabled: !!userId, // Only run if userId is provided
  });
};
