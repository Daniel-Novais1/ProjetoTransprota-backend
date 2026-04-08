import React, { useState } from 'react';
import { api } from '../api/client';
import { AlertCircle, MapPin, Send } from 'lucide-react';

function Reports() {
  const [formData, setFormData] = useState({
    user_id: localStorage.getItem('user_id') || '',
    bus_line: '',
    bus_id: '',
    type: 'Lotado',
    latitude: '',
    longitude: '',
    evidence_url: '',
  });

  const [loading, setLoading] = useState(false);
  const [sucesso, setSucesso] = useState(null);
  const [erro, setErro] = useState(null);
  const [mostrarTrust, setMostrarTrust] = useState(false);

  const tipos = ['Lotado', 'Atrasado', 'Não Parou', 'Ar Estragado', 'Sujo'];

  const obterLocalizacao = () => {
    if (navigator.geolocation) {
      navigator.geolocation.getCurrentPosition((position) => {
        setFormData((prev) => ({
          ...prev,
          latitude: position.coords.latitude.toString(),
          longitude: position.coords.longitude.toString(),
        }));
      });
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setErro(null);
    setSucesso(null);

    try {
      const response = await api.submitReport({
        ...formData,
        latitude: parseFloat(formData.latitude),
        longitude: parseFloat(formData.longitude),
      });

      setSucesso(response.data);
      setMostrarTrust(true);
      localStorage.setItem('user_id', formData.user_id);

      // Resetar formulário
      setFormData({
        user_id: formData.user_id,
        bus_line: '',
        bus_id: '',
        type: 'Lotado',
        latitude: '',
        longitude: '',
        evidence_url: '',
      });

      setTimeout(() => setMostrarTrust(false), 5000);
    } catch (error) {
      setErro(error.response?.data?.error || 'Erro ao submeter denúncia');
    } finally {
      setLoading(false);
    }
  };

  const getTrustLevel = (score) => {
    if (score <= 20) return { level: 'Suspeito', color: 'bg-red-100 text-red-700' };
    if (score <= 80) return { level: 'Cidadão', color: 'bg-blue-100 text-blue-700' };
    return { level: 'Fiscal da Galera', color: 'bg-green-100 text-green-700' };
  };

  return (
    <div className="max-w-2xl mx-auto">
      <div className="bg-white rounded-lg shadow-lg p-8">
        <h1 className="text-3xl font-bold text-gray-800 mb-2">🚨 Denúncias Colaborativas</h1>
        <p className="text-gray-600 mb-6">
          Ajude a comunidade a denunciar problemas nos ônibus de Goiânia
        </p>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* ID do Usuário */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Seu ID de Usuário
            </label>
            <input
              type="text"
              value={formData.user_id}
              onChange={(e) =>
                setFormData({ ...formData, user_id: e.target.value })
              }
              placeholder="Seu identificador único"
              required
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
            />
          </div>

          {/* Linha e Ônibus */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Número da Linha
              </label>
              <input
                type="text"
                value={formData.bus_line}
                onChange={(e) =>
                  setFormData({ ...formData, bus_line: e.target.value })
                }
                placeholder="Ex: 101"
                required
                maxLength="10"
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                ID do Ônibus
              </label>
              <input
                type="text"
                value={formData.bus_id}
                onChange={(e) =>
                  setFormData({ ...formData, bus_id: e.target.value })
                }
                placeholder="Ex: BUS-001"
                required
                maxLength="50"
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
              />
            </div>
          </div>

          {/* Tipo de Problema */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Tipo de Problema
            </label>
            <select
              value={formData.type}
              onChange={(e) =>
                setFormData({ ...formData, type: e.target.value })
              }
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
            >
              {tipos.map((tipo) => (
                <option key={tipo} value={tipo}>
                  {tipo}
                </option>
              ))}
            </select>
          </div>

          {/* Localização */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Localização
            </label>
            <div className="grid grid-cols-2 gap-4 mb-2">
              <input
                type="number"
                step="0.0001"
                value={formData.latitude}
                onChange={(e) =>
                  setFormData({ ...formData, latitude: e.target.value })
                }
                placeholder="Latitude"
                required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
              />
              <input
                type="number"
                step="0.0001"
                value={formData.longitude}
                onChange={(e) =>
                  setFormData({ ...formData, longitude: e.target.value })
                }
                placeholder="Longitude"
                required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
              />
            </div>
            <button
              type="button"
              onClick={obterLocalizacao}
              className="text-indigo-600 hover:text-indigo-700 font-semibold flex items-center gap-2"
            >
              <MapPin className="w-4 h-4" />
              Usar minha localização
            </button>
          </div>

          {/* Evidência */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              URL de Evidência (foto/vídeo) - Opcional
            </label>
            <input
              type="url"
              value={formData.evidence_url}
              onChange={(e) =>
                setFormData({ ...formData, evidence_url: e.target.value })
              }
              placeholder="https://example.com/foto.jpg"
              maxLength="2048"
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
            />
          </div>

          {erro && (
            <div className="bg-red-50 border-l-4 border-red-400 p-4 flex items-center gap-2">
              <AlertCircle className="w-5 h-5 text-red-400" />
              <p className="text-red-700">{erro}</p>
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-indigo-600 hover:bg-indigo-700 disabled:bg-gray-400 text-white font-semibold py-3 rounded-lg transition flex items-center justify-center gap-2"
          >
            <Send className="w-5 h-5" />
            {loading ? 'Enviando...' : 'Enviar Denúncia'}
          </button>
        </form>

        {mostrarTrust && sucesso && (
          <div className={`mt-6 p-4 rounded-lg border-l-4 ${getTrustLevel(sucesso.trust_score).color}`}>
            <p className="font-semibold mb-2">✓ Denúncia enviada com sucesso!</p>
            <p className="text-sm">
              Seu Trust Score: <strong>{sucesso.trust_score}/100</strong> -{' '}
              {getTrustLevel(sucesso.trust_score).level}
            </p>
          </div>
        )}
      </div>

      {/* Informações sobre Trust Score */}
      <div className="bg-blue-50 rounded-lg shadow p-8 mt-8 border border-blue-200">
        <h2 className="text-xl font-bold text-blue-900 mb-4">📊 Como funciona o Trust Score?</h2>
        <div className="space-y-3 text-sm text-blue-800">
          <p>✓ <strong>+5 pontos</strong> cada vez que sua denúncia é confirmada pela comunidade</p>
          <p>✗ <strong>-15 pontos</strong> se sua denúncia for marcada como spam</p>
          <p>📸 <strong>+10 pontos</strong> bônus por denúncias com evidência (foto/vídeo)</p>
          <p className="mt-4">
            <strong>Níveis:</strong> Suspeito (0-20) → Cidadão (21-80) → Fiscal da Galera (81-100)
          </p>
        </div>
      </div>
    </div>
  );
}

export default Reports;
