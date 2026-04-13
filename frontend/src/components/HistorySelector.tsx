import React, { useState } from 'react';
import { Calendar, Clock, Search } from 'lucide-react';

interface HistorySelectorProps {
  onSearch: (deviceId: string, startTime: string, endTime: string) => void;
  loading?: boolean;
}

export default function HistorySelector({ onSearch, loading = false }: HistorySelectorProps) {
  const [deviceId, setDeviceId] = useState('');
  const [startTime, setStartTime] = useState('');
  const [endTime, setEndTime] = useState('');

  const handleSearch = () => {
    if (!deviceId) {
      alert('Por favor, insira o ID do dispositivo');
      return;
    }

    // Se não especificado, usar padrão (última hora)
    const start = startTime || new Date(Date.now() - 60 * 60 * 1000).toISOString().slice(0, 16);
    const end = endTime || new Date().toISOString().slice(0, 16);

    onSearch(deviceId, start, end);
  };

  const handleQuickSelect = (hours: number) => {
    const end = new Date();
    const start = new Date(Date.now() - hours * 60 * 60 * 1000);
    
    setStartTime(start.toISOString().slice(0, 16));
    setEndTime(end.toISOString().slice(0, 16));
  };

  return (
    <div className="bg-white rounded-lg shadow-lg p-6 mb-4">
      <h3 className="text-lg font-bold text-gray-800 mb-4 flex items-center gap-2">
        <Calendar className="w-5 h-5" />
        Histórico de Trajetória
      </h3>

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            ID do Dispositivo
          </label>
          <input
            type="text"
            value={deviceId}
            onChange={(e) => setDeviceId(e.target.value)}
            placeholder="Ex: bus-001"
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Início
            </label>
            <input
              type="datetime-local"
              value={startTime}
              onChange={(e) => setStartTime(e.target.value)}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Fim
            </label>
            <input
              type="datetime-local"
              value={endTime}
              onChange={(e) => setEndTime(e.target.value)}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
        </div>

        <div className="flex gap-2">
          <button
            onClick={() => handleQuickSelect(1)}
            className="px-3 py-1 text-sm bg-gray-100 hover:bg-gray-200 rounded-lg transition-colors"
          >
            1h
          </button>
          <button
            onClick={() => handleQuickSelect(6)}
            className="px-3 py-1 text-sm bg-gray-100 hover:bg-gray-200 rounded-lg transition-colors"
          >
            6h
          </button>
          <button
            onClick={() => handleQuickSelect(24)}
            className="px-3 py-1 text-sm bg-gray-100 hover:bg-gray-200 rounded-lg transition-colors"
          >
            24h
          </button>
        </div>

        <button
          onClick={handleSearch}
          disabled={loading}
          className="w-full bg-blue-600 text-white py-3 rounded-lg font-medium hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
        >
          {loading ? (
            <>
              <Clock className="w-4 h-4 animate-spin" />
              Buscando...
            </>
          ) : (
            <>
              <Search className="w-4 h-4" />
              Buscar Histórico
            </>
          )}
        </button>
      </div>
    </div>
  );
}
