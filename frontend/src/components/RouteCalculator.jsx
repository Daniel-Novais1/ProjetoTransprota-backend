import React, { useState } from 'react';
import { api } from '../api/client';
import { MapPin, Clock, Zap } from 'lucide-react';

function RouteCalculator() {
  const [origem, setOrigem] = useState('');
  const [destino, setDestino] = useState('');
  const [rota, setRota] = useState(null);
  const [loading, setLoading] = useState(false);
  const [erro, setErro] = useState(null);

  const handleCalcular = async (e) => {
    e.preventDefault();
    setLoading(true);
    setErro(null);
    setRota(null);

    try {
      const response = await api.calcularRota(origem, destino);
      setRota(response.data);
    } catch (error) {
      setErro(error.response?.data?.error || 'Erro ao calcular rota. Tente novamente.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto">
      <div className="bg-white rounded-lg shadow-lg p-8 mb-8">
        <h1 className="text-3xl font-bold text-gray-800 mb-6">📍 Planejador de Rotas</h1>

        <form onSubmit={handleCalcular} className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                De onde você sai?
              </label>
              <input
                type="text"
                value={origem}
                onChange={(e) => setOrigem(e.target.value)}
                placeholder="Ex: Vila Pedroso"
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Para onde você vai?
              </label>
              <input
                type="text"
                value={destino}
                onChange={(e) => setDestino(e.target.value)}
                placeholder="Ex: UFG"
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                required
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-indigo-600 hover:bg-indigo-700 disabled:bg-gray-400 text-white font-semibold py-3 rounded-lg transition"
          >
            {loading ? 'Calculando...' : 'Calcular Rota'}
          </button>
        </form>
      </div>

      {erro && (
        <div className="bg-red-50 border-l-4 border-red-400 p-4 mb-8">
          <p className="text-red-700">{erro}</p>
        </div>
      )}

      {rota && (
        <div className="bg-white rounded-lg shadow-lg p-8">
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-2xl font-bold text-gray-800">
              {rota.origem} → {rota.destino}
            </h2>
            <span className={`px-4 py-2 rounded-full text-sm font-semibold ${
              rota.cached ? 'bg-green-100 text-green-800' : 'bg-blue-100 text-blue-800'
            }`}>
              {rota.cached ? '⚡ Cacheado' : '🔄 Recalculado'}
            </span>
          </div>

          <div className="space-y-6">
            {rota.steps.map((step, index) => (
              <div key={index} className="border-l-4 border-indigo-600 pl-6 py-4 bg-gray-50 rounded">
                <div className="flex items-center justify-between mb-3">
                  <h3 className="text-lg font-bold text-indigo-600">
                    Linha {step.numero_linha}
                  </h3>
                  <div className="flex items-center gap-2 text-gray-700">
                    <Clock className="w-5 h-5" />
                    <span className="font-semibold">{step.tempo_total_minutos} min</span>
                  </div>
                </div>

                <p className="text-gray-700 mb-3">{step.nome_linha}</p>

                <div className="bg-white p-3 rounded border border-gray-200">
                  <p className="text-sm text-gray-600">
                    {step.paradas.join(' → ')}
                  </p>
                </div>
              </div>
            ))}
          </div>

          <div className="mt-8 p-4 bg-indigo-50 rounded-lg border border-indigo-200">
            <p className="text-indigo-900 font-semibold">
              ⏱️ Tempo total estimado:{' '}
              <span className="text-2xl text-indigo-600">
                {rota.steps.reduce((total, step) => total + step.tempo_total_minutos, 0)} minutos
              </span>
            </p>
          </div>
        </div>
      )}
    </div>
  );
}

export default RouteCalculator;
