import React, { useEffect, useState } from "react";
import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import { api } from "@/api/client";
import { HealthResponse } from "@/types";
import Navigation from "@/components/Navigation";
import RoutePlanner from "@/components/RoutePlanner";
import BusMap from "@/components/BusMap";
import ArrivalBoard from "@/components/ArrivalBoard";
import Reports from "@/components/Reports";
import { AlertCircle } from "lucide-react";
import "./App.css";

export default function App() {
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const checkHealth = async () => {
      try {
        await api.getHealth();
        setError(null);
      } catch (err: any) {
        setError(api.extractErrorMessage(err));
      }
    };

    checkHealth();
  }, []);

  return (
    <Router>
      <div className="min-h-screen bg-gray-50 flex flex-col">
        <Navigation />
        {error && (
          <div className="bg-red-50 border-b border-red-200 px-4 py-3 flex items-start gap-3">
            <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" />
            <div>
              <p className="font-semibold text-red-900">⚠️ Erro de Conexão</p>
              <p className="text-sm text-red-700">{error}</p>
            </div>
          </div>
        )}

        <main className="flex-1 py-8">
          <Routes>
            <Route
              path="/"
              element={
                <div className="max-w-7xl mx-auto px-4">
                  <div className="mb-8 text-center">
                    <h1 className="text-4xl font-bold text-gray-800 mb-2">
                      🚌 TranspRota
                    </h1>
                    <p className="text-gray-600">
                      Planeje sua viagem, rastreie ônibus e reporte problemas em
                      tempo real
                    </p>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <RoutePlanner />
                    <BusMap />
                  </div>
                </div>
              }
            />

            <Route
              path="/planejar"
              element={
                <div className="max-w-7xl mx-auto px-4">
                  <RoutePlanner />
                </div>
              }
            />

            <Route
              path="/rastrear"
              element={
                <div className="max-w-7xl mx-auto px-4">
                  <BusMap />
                </div>
              }
            />

            <Route
              path="/chegadas"
              element={
                <div className="max-w-7xl mx-auto px-4">
                  <ArrivalBoard latitude={-15.8} longitude={-48.0} />
                </div>
              }
            />

            <Route
              path="/denuncias"
              element={
                <div className="max-w-7xl mx-auto px-4">
                  <Reports />
                </div>
              }
            />

            <Route
              path="*"
              element={
                <div className="max-w-7xl mx-auto px-4 text-center py-12">
                  <h2 className="text-2xl font-bold text-gray-800 mb-4">
                    404 - Página Não Encontrada
                  </h2>
                  <a href="/" className="text-blue-600 hover:underline">
                    ← Voltar para Home
                  </a>
                </div>
              }
            />
          </Routes>
        </main>

        <footer className="bg-gray-800 text-gray-300 text-center py-4 text-sm">
          TranspRota v1.0 | Precisão em Tempo Real
        </footer>
      </div>
    </Router>
  );
}
