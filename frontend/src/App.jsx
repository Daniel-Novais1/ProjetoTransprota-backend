import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import RouteCalculator from './components/RouteCalculator';
import BusTracker from './components/BusTracker';
import Reports from './components/Reports';
import Navigation from './components/Navigation';
import { api } from './api/client';
import './App.css';

function App() {
  const [health, setHealth] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    checkHealth();
  }, []);

  const checkHealth = async () => {
    try {
      const response = await api.getHealth();
      setHealth(response.data);
    } catch (error) {
      console.error('Erro ao verificar saúde da API:', error);
      setHealth({ status: 'offline' });
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen bg-gradient-to-br from-blue-50 to-indigo-100">
        <div className="text-center">
          <div className="mb-4">
            <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600"></div>
          </div>
          <h1 className="text-2xl font-bold text-gray-800 mb-2">TranspRota</h1>
          <p className="text-gray-600">Carregando...</p>
        </div>
      </div>
    );
  }

  return (
    <Router>
      <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100">
        <Navigation health={health} />
        
        {health?.status !== 'ok' && (
          <div className="bg-yellow-50 border-l-4 border-yellow-400 p-4 mx-4 mt-4">
            <p className="text-yellow-700">
              ⚠️ API com status: <strong>{health?.status}</strong>
              {health?.database !== 'ok' && ' (Banco de dados com problemas)'}
              {health?.redis !== 'ok' && ' (Redis com problemas)'}
            </p>
          </div>
        )}

        <main className="container mx-auto px-4 py-8">
          <Routes>
            <Route path="/" element={<RouteCalculator />} />
            <Route path="/rastrear" element={<BusTracker />} />
            <Route path="/denuncias" element={<Reports />} />
          </Routes>
        </main>
      </div>
    </Router>
  );
}

export default App;
