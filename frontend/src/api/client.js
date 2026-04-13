import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

const client = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

export const api = {
  // Linhas
  getLinhas: () => client.get('/linhas'),

  // Terminais
  getTerminais: () => client.get('/terminais'),

  // Rotas
  calcularRota: (origem, destino) =>
    client.get('/planejar', { params: { origem, destino } }),

  // GPS
  getBusLocation: (busId) => client.get(`/gps/${busId}`),
  getBusStatus: (busId) => client.get(`/gps/${busId}/status`),
  updateBusLocation: (data) => client.post('/gps', data),

  // Denúncias
  submitReport: (data) => client.post('/denuncias', data),
  getNearbyReports: (lat, lon, radius = 5000) =>
    client.get('/denuncias', { params: { lat, lon, radius } }),

  // Sistema
  getHealth: () => client.get('/health'),
  getMetrics: () => client.get('/metrics'),
};

export default client;
