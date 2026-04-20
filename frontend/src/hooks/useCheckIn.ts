import { useMutation } from '@tanstack/react-query';
import confetti from 'canvas-confetti';

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

interface CheckInRequest {
  deviceHash: string;
  routeId: string;
  lat: number;
  lng: number;
}

interface CheckInResponse {
  status: 'success';
  points: number;
  newStreak: number;
}

export const useCheckIn = () => {
  return useMutation({
    mutationFn: async (data: CheckInRequest): Promise<CheckInResponse> => {
      const response = await fetch(`${API_BASE}/api/v1/telemetry/checkin`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
      });
      
      if (!response.ok) {
        throw new Error('Check-in failed');
      }
      
      return response.json();
    },
    onSuccess: () => {
      // Trigger confetti animation
      confetti({
        particleCount: 100,
        spread: 70,
        origin: { y: 0.6 },
        colors: ['#4ae176', '#adc6ff', '#ffb786'],
      });
    },
  });
};
