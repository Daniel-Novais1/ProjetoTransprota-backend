import React, { useState } from "react";
import { api } from "@/api/client";
import { SubmeterDenunciaRequest, TipoDenuncia } from "@/types";
import { AlertCircle, MapPin, Send, CheckCircle } from "lucide-react";

export const Reports: React.FC = () => {
  const [userId] = useState(() => {
    let id = localStorage.getItem("transprota_user_id");
    if (!id) {
      id = `user_${Date.now()}`;
      localStorage.setItem("transprota_user_id", id);
    }
    return id;
  });

  const [form, setForm] = useState<SubmeterDenunciaRequest>({
    user_id: userId,
    bus_line: "",
    bus_id: "",
    type: TipoDenuncia.Atrasado,
    latitude: 0,
    longitude: 0,
  });

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [trustScore, setTrustScore] = useState<{ score: number; level: string } | null>(null);

  const useMyLocation = async () => {
    if ("geolocation" in navigator) {
      navigator.geolocation.getCurrentPosition(
        (position) => {
          setForm((prev) => ({
            ...prev,
            latitude: position.coords.latitude,
            longitude: position.coords.longitude,
          }));
        },
        (error) => {
          setError(`Erro ao obter localização: ${error.message}`);
        }
      );
    }
  };

  const submitReport = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!form.bus_line || !form.bus_id) {
      setError("Linha e ID do ônibus são obrigatórios");
      return;
    }

    if (form.latitude === 0 || form.longitude === 0) {
      setError("Localização é obrigatória. Use o botão 'Minha Localização'");
      return;
    }

    setLoading(true);
    setError(null);
    setSuccess(false);

    try {
      const response = await api.submitReport(form);

      const score = {
        score: response.trust_score,
        level:
          response.trust_score <= 20
            ? "Suspeito"
            : response.trust_score <= 80
            ? "Cidadão"
            : "Fiscal da Galera",
      };
      setTrustScore(score);
      setSuccess(true);

      setForm((prev) => ({
        ...prev,
        bus_line: "",
        bus_id: "",
        type: TipoDenuncia.Atrasado,
        evidence_url: "",
        latitude: 0,
        longitude: 0,
      }));

      setTimeout(() => setSuccess(false), 5000);
    } catch (err: any) {
      setError(api.extractErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="w-full max-w-2xl mx-auto p-6 bg-white rounded-lg shadow-md">
      <h2 className="text-2xl font-bold mb-4 flex items-center gap-2">
        <AlertCircle className="w-6 h-6 text-orange-600" />
        Denunciar Problema
      </h2>

      <div className="mb-4 p-3 bg-blue-50 border border-blue-200 rounded-lg text-sm">
        <p className="font-semibold text-blue-900 mb-2">📊 Como funciona:</p>
        <ul className="text-blue-800 space-y-1 text-xs">
          <li>✅ +5 pontos por denúncia confirmada</li>
          <li>❌ -15 pontos por denúncia falsa (spam)</li>
          <li>📸 +10 pontos por completar com evidência</li>
          <li>
            Níveis: Suspeito (0-20) | Cidadão (21-80) | Fiscal da Galera (81-100)
          </li>
        </ul>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg flex gap-2">
          <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0" />
          <span className="text-red-700 text-sm">{error}</span>
        </div>
      )}

      {success && trustScore && (
        <div className="mb-4 p-4 bg-green-50 border border-green-200 rounded-lg">
          <div className="flex items-start gap-2 mb-3">
            <CheckCircle className="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p className="font-semibold text-green-900">✅ Denúncia enviada!</p>
              <p className="text-sm text-green-800 mt-1">
                Seu score agora é: {trustScore.score}/100 ({trustScore.level})
              </p>
            </div>
          </div>
        </div>
      )}

      <form onSubmit={submitReport} className="space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Linha do Ônibus
            </label>
            <input
              type="text"
              value={form.bus_line}
              onChange={(e) => setForm((prev) => ({ ...prev, bus_line: e.target.value }))}
              placeholder="ex: 101"
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              ID do Ônibus
            </label>
            <input
              type="text"
              value={form.bus_id}
              onChange={(e) => setForm((prev) => ({ ...prev, bus_id: e.target.value }))}
              placeholder="ex: BUS_001"
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
            />
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Tipo de Problema
          </label>
          <select
            value={form.type}
            onChange={(e) => setForm((prev) => ({ ...prev, type: e.target.value as TipoDenuncia }))}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
          >
            <option value={TipoDenuncia.Lotado}>Lotado</option>
            <option value={TipoDenuncia.Atrasado}>Atrasado</option>
            <option value={TipoDenuncia.NaoParou}>Não Parou</option>
            <option value={TipoDenuncia.ArEstragado}>Ar Estragado</option>
            <option value={TipoDenuncia.Sujo}>Sujo</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Evidência (URL opcional)
          </label>
          <input
            type="url"
            value={form.evidence_url || ""}
            onChange={(e) => setForm((prev) => ({ ...prev, evidence_url: e.target.value }))}
            placeholder="https://..."
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
          />
        </div>

        <div className="flex gap-2">
          <button
            type="button"
            onClick={useMyLocation}
            className="flex-1 px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition flex items-center justify-center gap-2"
          >
            <MapPin className="w-4 h-4" />
            Minha Localização
          </button>
          <button
            type="submit"
            disabled={loading}
            className="flex-1 px-4 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 disabled:opacity-50 transition flex items-center justify-center gap-2"
          >
            {loading ? (
              "Enviando..."
            ) : (
              <>
                <Send className="w-4 h-4" />
                Enviar Denúncia
              </>
            )}
          </button>
        </div>
      </form>
    </div>
  );
};

export default Reports;
              type="button"
              onClick={useMyLocation}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition flex items-center gap-1"
            >
              <MapPin className="w-4 h-4" />
              Minha Loc
            </button>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            URL de Evidência (foto, etc)
          </label>
          <input
            type="url"
            value={form.evidence_url || ""}
            onChange={(e) =>
              setForm((prev) => ({ ...prev, evidence_url: e.target.value }))
            }
            placeholder="https://..."
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-500"
          />
        </div>

        <button
          type="submit"
          disabled={loading}
          className="w-full px-6 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 disabled:opacity-50 transition flex items-center justify-center gap-2 font-semibold"
        >
          <Send className="w-4 h-4" />
          {loading ? "Enviando..." : "Enviar Denúncia"}
        </button>
      </form>
    </div>
  );
};

export default Reports;
