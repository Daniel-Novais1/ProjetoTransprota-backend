import React from "react";
import { RouteResponse } from "@/types";
import { Clock, MapPin } from "lucide-react";

interface ArrivalBoardProps {
  route: RouteResponse;
  totalTime: number;
}

export const ArrivalBoard: React.FC<ArrivalBoardProps> = ({ route, totalTime }) => {
  if (!route || route.steps.length === 0) {
    return null;
  }

  const firstStep = route.steps[0];

  return (
    <div className="bg-white p-4 rounded-lg border border-gray-200 shadow-sm">
      <h3 className="text-lg font-semibold mb-3 flex items-center gap-2">
        <Clock className="w-5 h-5 text-green-600" />
        Tempo Estimado de Chegada
      </h3>

      <div className="space-y-3">
        <div className="flex items-center justify-between p-3 bg-green-50 rounded-lg border border-green-200">
          <div className="flex items-center gap-3">
            <MapPin className="w-5 h-5 text-green-600" />
            <div>
              <p className="font-medium text-gray-900">
                Linha {firstStep.numero_linha} - {firstStep.nome_linha}
              </p>
              <p className="text-sm text-gray-600">
                {route.origem} → {route.destino}
              </p>
            </div>
          </div>
          <div className="text-right">
            <p className="text-2xl font-bold text-green-600">{totalTime} min</p>
            <p className="text-xs text-gray-500">estimativa</p>
          </div>
        </div>

        <div className="text-xs text-gray-500 text-center">
          Rota calculada em tempo real • Dados sujeitos a variações no trânsito
        </div>
      </div>
    </div>
  );
};
      ];

      setArrivals(simulatedArrivals);
    } catch (err: any) {
      setError(api.extractErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchNearbyArrivals();
  }, [latitude, longitude]);

  return (
    <div className="w-full max-w-4xl mx-auto space-y-6">
      <div className="bg-white rounded-lg shadow-md overflow-hidden">
        <div className="bg-gradient-to-r from-green-600 to-teal-600 text-white p-4">
          <h2 className="text-2xl font-bold">📋 Quadro de Chegadas</h2>
          <p className="text-sm text-green-100 mt-1">
            Próximas linhas em sua localização
          </p>
        </div>

        {error && (
          <div className="m-4 p-3 bg-red-50 border border-red-200 rounded-lg flex gap-2">
            <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" />
            <span className="text-red-700 text-sm">{error}</span>
          </div>
        )}

        {loading && (
          <div className="flex justify-center items-center py-12">
            <Loader className="w-8 h-8 text-blue-600 animate-spin" />
          </div>
        )}

        {arrivals.length > 0 && !loading && (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-100 border-b border-gray-300">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-700">
                    Linha
                  </th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-700">
                    Terminal
                  </th>
                  <th className="px-4 py-3 text-center text-sm font-semibold text-gray-700">
                    ETA
                  </th>
                  <th className="px-4 py-3 text-center text-sm font-semibold text-gray-700">
                    Distância
                  </th>
                  <th className="px-4 py-3 text-center text-sm font-semibold text-gray-700">
                    Status
                  </th>
                </tr>
              </thead>
              <tbody>
                {arrivals.map((arrival, idx) => (
                  <tr
                    key={idx}
                    className={idx % 2 === 0 ? "bg-white" : "bg-gray-50"}
                  >
                    <td className="px-4 py-4 text-lg font-bold text-blue-600">
                      {arrival.linha}
                    </td>
                    <td className="px-4 py-4 text-gray-800">
                      {arrival.proximoTerminal}
                    </td>
                    <td className="px-4 py-4 text-center">
                      <span className="inline-flex items-center gap-1 text-base font-semibold text-green-600">
                        ⏱️ {arrival.etaMinutos} min
                      </span>
                    </td>
                    <td className="px-4 py-4 text-center text-sm text-gray-600">
                      {(arrival.distanciaMetros / 1000).toFixed(2)} km
                    </td>
                    <td className="px-4 py-4 text-center">
                      <StatusBadge
                        status={arrival.status as "Em trânsito" | "No Terminal"}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {arrivals.length === 0 && !loading && !error && (
          <div className="text-center py-8 text-gray-500">
            Nenhuma linha próxima no momento
          </div>
        )}
      </div>

      {denuncias.length > 0 && (
        <div className="bg-white rounded-lg shadow-md p-4">
          <h3 className="text-lg font-bold mb-3 flex items-center gap-2">
            ⚠️ Denúncias Ativas Próximas ({denuncias.length})
          </h3>
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {denuncias.map((denuncia) => (
              <div
                key={denuncia.id}
                className="p-3 bg-yellow-50 border border-yellow-200 rounded-lg text-sm"
              >
                <p className="font-semibold text-yellow-900">
                  Linha {denuncia.bus_line} | {denuncia.type}
                </p>
                <p className="text-yellow-800 text-xs mt-1">
                  Usuário score: {denuncia.trust_score}/100 •{" "}
                  {new Date(denuncia.timestamp).toLocaleTimeString("pt-BR")}
                </p>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default ArrivalBoard;
