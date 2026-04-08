import React, { useState, useEffect } from 'react';
import { api } from '../api/client';
import { MapPin, Navigation, AlertCircle } from 'lucide-react';

function BusTracker() {
  const [busId, setBusId] = useState('');
  const [busData, setBusData] = useState(null);
  const [status, setStatus] = useState(null);
  const [loading, setLoading] = useState(false);
  const [erro, setErro] = useState(null);
  const [autoRefresh, setAutoRefresh] = useState(false);

  useEffect(() => {
    if (!autoRefresh) return;

    const interval = setInterval(() => {
      if (busId) rastrear();
    }, 5000);

    return () => clearInterval(interval);
  }, [autoRefresh, busId]);

  const rastrear = async () => {
    if (!busId.trim()) return;

    setLoading(true);
    setErro(null);

    try {
      const [locationResponse, statusResponse] = await Promise.all([
        api.getBusLocation(busId),
        api.getBusStatus(busId),
      ]);

      setBusData(locationResponse.data);
      setStatus(statusResponse.data);
    } catch (error) {
      setErro(error.response?.data?.error || 'Ônibus não encontrado');
      setBusData(null);
      setStatus(null);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto">
      <div className="bg-white rounded-lg shadow-lg p-8 mb-8">
        <h1 className="text-3xl font-bold text-gray-800 mb-6">📍 Rastreador de Ônibus</h1>

        <div className="space-y-6">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              ID do Ônibus
            </label>
            <div className="flex gap-2">
              <input
                type="text"
                value={busId}
                onChange={(e) => setBusId(e.target.value)}
                placeholder="Ex: BUS-001"
                className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
              />
              <button
                onClick={rastrear}
                disabled={loading}
                className="bg-indigo-600 hover:bg-indigo-700 disabled:bg-gray-400 text-white font-semibold px-6 py-2 rounded-lg transition"
              >
                {loading ? 'Rastreando...' : 'Rastrear'}
              </button>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="autoRefresh"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
              className="w-4 h-4 text-indigo-600 rounded"
            />
            <label htmlFor="autoRefresh" className="text-gray-700">
              Atualizar automaticamente a cada 5 segundos
            </label>
          </div>
        </div>
      </div>

      {erro && (
        <div className="bg-red-50 border-l-4 border-red-400 p-4 mb-8 flex items-center gap-2">
          <AlertCircle className="w-5 h-5 text-red-400" />
          <p className="text-red-700">{erro}</p>
        </div>
      )}

      {busData && status && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
          {/* Localização Atual */}
          <div className="bg-white rounded-lg shadow-lg p-8">
            <h2 className="text-xl font-bold text-gray-800 mb-6 flex items-center gap-2">
              <MapPin className="w-6 h-6 text-indigo-600" />
              Localização Atual
            </h2>

            <div className="space-y-4">
              <div>
                <p className="text-sm text-gray-600">Ônibus</p>
                <p className="text-2xl font-bold text-gray-800">{busData.bus_id}</p>
              </div>

              <div className="bg-indigo-50 p-4 rounded-lg border border-indigo-200">
                <p className="text-sm text-gray-600">Coordenadas</p>
                <p className="font-mono text-sm text-indigo-600 font-semibold">
                  {busData.latitude.toFixed(4)}, {busData.longitude.toFixed(4)}
                </p>
              </div>

              <div>
                <p className="text-sm text-gray-600">Atualizado em</p>
                <p className="text-gray-800">
                  {new Date(busData.timestamp).toLocaleTimeString('pt-BR')}
                </p>
              </div>

              <div className="mt-4 p-4 bg-blue-50 rounded-lg border border-blue-200">
                <p className="text-sm text-blue-600 font-semibold">
                  💡 Dica: Abra esse link no Google Maps para ver a localização em tempo real
                </p>
                <a
                  href={`https://maps.google.com/?q=${busData.latitude},${busData.longitude}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="mt-2 inline-block text-indigo-600 hover:text-indigo-700 font-semibold"
                >
                  Ver no Google Maps →
                </a>
              </div>
            </div>
          </div>

          {/* Status */}
          <div className="bg-white rounded-lg shadow-lg p-8">
            <h2 className="text-xl font-bold text-gray-800 mb-6 flex items-center gap-2">
              <Navigation className="w-6 h-6 text-indigo-600" />
              Status
            </h2>

            <div className="space-y-4">
              <div className="p-4 rounded-lg bg-gray-50 border border-gray-200">
                <p className="text-sm text-gray-600 mb-1">Estado</p>
                <p className={`text-2xl font-bold ${
                  status.status === 'No Terminal'
                    ? 'text-green-600'
                    : 'text-blue-600'
                }`}>
                  {status.status}
                </p>
              </div>

              {status.terminal && (
                <div className="p-4 rounded-lg bg-green-50 border border-green-200">
                  <p className="text-sm text-gray-600 mb-1">Terminal Próximo</p>
                  <p className="text-lg font-semibold text-green-700">{status.terminal}</p>
                  <p className="text-sm text-green-600 mt-2">
                    Distância: {status.distancia_metros.toFixed(0)} metros
                  </p>
                </div>
              )}

              <div className="mt-6 p-4 bg-indigo-50 rounded-lg border border-indigo-200">
                <p className="text-sm text-indigo-600">
                  {status.status === 'No Terminal'
                    ? '✓ Ônibus chegou no terminal!'
                    : '📍 Ônibus em circulação'}
                </p>
              </div>
            </div>
          </div>
        </div>
      )}

      {!busData && !erro && (
        <div className="text-center text-gray-500 py-12">
          <MapPin className="w-16 h-16 mx-auto mb-4 opacity-30" />
          <p>Insira um ID de ônibus para começar o rastreamento</p>
        </div>
      )}
    </div>
  );
}

export default BusTracker;
