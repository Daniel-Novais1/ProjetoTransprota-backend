import React, { useState } from "react";
import { api } from "@/api/client";
import { RouteResponse } from "@/types";
import { Navigation2, Clock, AlertCircle, Zap } from "lucide-react";
import { BusMap } from "./BusMap";
import { ArrivalBoard } from "./ArrivalBoard";

export const RoutePlanner: React.FC = () => {
  const [origem, setOrigem] = useState("Vila Pedroso");
  const [destino, setDestino] = useState("UFG");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [route, setRoute] = useState<RouteResponse | null>(null);

  const planejadorRota = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!origem.trim() || !destino.trim()) {
      setError("Origem e destino são obrigatórios");
      return;
    }

    setLoading(true);
    setError(null);
    setRoute(null);

    try {
      const result = await api.calcularRota({ origem, destino });
      setRoute(result);
    } catch (err: any) {
      const errorMsg = api.extractErrorMessage(err);
      if (errorMsg.includes("Erro ao calcular rota") || errorMsg.includes("Nenhuma rota encontrada")) {
        setError("Ops! O Eixo Anhanguera parece congestionado ou a rota não foi encontrada. Tente novamente mais tarde.");
      } else {
        setError("Erro de conexão com o servidor. Verifique sua internet.");
      }
    } finally {
      setLoading(false);
    }
  };

  const calculateTotalTime = (): number => {
    if (!route) return 0;
    return route.steps.reduce((sum, step) => sum + step.tempo_total_minutos, 0);
  };

  return (
    <div className="w-full max-w-4xl mx-auto p-6 bg-white rounded-lg shadow-md">
      <h2 className="text-2xl font-bold mb-4 flex items-center gap-2">
        <Navigation2 className="w-6 h-6 text-purple-600" />
        Planejador de Rota
      </h2>

      <form onSubmit={planejadorRota} className="space-y-4 mb-6">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Origem
            </label>
            <input
              type="text"
              value={origem}
              onChange={(e) => setOrigem(e.target.value)}
              placeholder="Ex: Vila Pedroso"
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-purple-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Destino
            </label>
            <input
              type="text"
              value={destino}
              onChange={(e) => setDestino(e.target.value)}
              placeholder="Ex: UFG"
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-purple-500"
            />
          </div>
        </div>

        <button
          type="submit"
          disabled={loading}
          className="w-full px-6 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 disabled:opacity-50 transition font-semibold"
        >
          {loading ? "Calculando..." : "Planejar Rota"}
        </button>
      </form>

      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg flex gap-2 items-start">
          <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" />
          <span className="text-red-700">{error}</span>
        </div>
      )}

      {route && (
        <div className="space-y-6">
          {/* Mapa com rota */}
          <BusMap route={route.steps[0]} center={[-16.686, -49.264]} />

          {/* Arrival Board */}
          <ArrivalBoard route={route} totalTime={calculateTotalTime()} />

          <div className="bg-gradient-to-r from-purple-50 to-blue-50 p-4 rounded-lg border border-purple-200">
            <div className="flex items-start justify-between mb-3">
              <div>
                <p className="text-sm text-gray-600">
                  {route.origem} → {route.destino}
                </p>
                <p className="text-xs text-gray-500 capitalize mt-1">
                  Tipo: {route.tipo === "direta" ? "✅ Rota Direta" : "🔄 Com Transferência"}
                </p>
              </div>
              <div className="text-right">
                <p className="text-lg font-bold text-purple-600 flex items-center gap-1">
                  <Clock className="w-4 h-4" />
                  {calculateTotalTime()} min
                </p>
                {route.cached && (
                  <p className="text-xs text-green-600 flex items-center gap-1 mt-1">
                    <Zap className="w-3 h-3" />
                    Do cache
                  </p>
                )}
              </div>
            </div>

            <div className="space-y-3">
              {route.steps.map((step, index) => (
                <div key={index} className="bg-white p-3 rounded border border-gray-200">
                  <div className="flex items-center justify-between mb-2">
                    <span className="font-semibold text-purple-700">
                      Linha {step.numero_linha} - {step.nome_linha}
                    </span>
                    <span className="text-sm text-gray-600">
                      {step.tempo_total_minutos} min
                    </span>
                  </div>
                  <div className="text-sm text-gray-700">
                    <span className="font-medium">Paradas:</span> {step.paradas.join(" → ")}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
                </p>
              </div>
              {route.cached && (
                <div className="flex items-center gap-1 text-yellow-600">
                  <Zap className="w-4 h-4" />
                  <span className="text-xs font-semibold">Cached</span>
                </div>
              )}
            </div>

            <div className="flex items-center gap-2 text-2xl font-bold text-purple-700">
              <Clock className="w-6 h-6" />
              {calculateTotalTime()} min
            </div>
          </div>

          <div className="space-y-3">
            {route.steps.map((step, idx) => (
              <div key={idx} className="border border-gray-200 rounded-lg overflow-hidden">
                <div className="bg-blue-50 px-4 py-2 border-b border-gray-200">
                  <p className="font-semibold text-blue-900">
                    Linha {step.numero_linha} - {step.nome_linha}
                  </p>
                  <p className="text-sm text-blue-700">
                    ⏱️ {step.tempo_total_minutos} minutos
                  </p>
                </div>
                <div className="px-4 py-3">
                  <p className="text-xs text-gray-600 mb-2 font-semibold">Itinerário:</p>
                  <ol className="space-y-1 text-sm">
                    {step.paradas.map((parada, i) => (
                      <li key={i} className="text-gray-700">
                        {i + 1}. {parada}
                      </li>
                    ))}
                  </ol>
                </div>
              </div>
            ))}
          </div>

          <div className="p-4 bg-green-50 border border-green-200 rounded-lg">
            <p className="text-sm font-semibold text-green-900">
              ✅ Total: {route.steps.length} linha{route.steps.length > 1 ? "s" : ""} | {calculateTotalTime()} min
            </p>
          </div>
        </div>
      )}

      {!route && !error && !loading && (
        <div className="text-center py-8 text-gray-500">
          Preencha origem e destino para calcular a melhor rota
        </div>
      )}
    </div>
  );
};

export default RoutePlanner;
