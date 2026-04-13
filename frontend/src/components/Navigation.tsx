import React, { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { api } from "@/api/client";
import { HealthResponse } from "@/types";
import { Bus, AlertCircle, Check } from "lucide-react";

export const Navigation: React.FC = () => {
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [isStale, setIsStale] = useState(false);

  useEffect(() => {
    const checkHealth = async () => {
      try {
        const data = await api.getHealth();
        setHealth(data);
        setIsStale(false);
      } catch (err) {
        setIsStale(true);
      }
    };

    checkHealth();
    const interval = setInterval(checkHealth, 30000);

    return () => clearInterval(interval);
  }, []);

  const healthColor =
    health?.status === "ok"
      ? "bg-green-100 text-green-800"
      : health?.status === "degraded"
      ? "bg-yellow-100 text-yellow-800"
      : "bg-red-100 text-red-800";

  return (
    <nav className="bg-gradient-to-r from-blue-600 to-blue-800 text-white shadow-md">
      <div className="max-w-7xl mx-auto px-4 py-4 flex items-center justify-between">
        <Link to="/" className="flex items-center gap-2 text-2xl font-bold hover:opacity-90">
          <Bus className="w-8 h-8" />
          TranspRota
        </Link>

        <div className="flex items-center gap-6">
          <Link to="/planejar" className="hover:opacity-80 transition">
            Planejar Rota
          </Link>
          <Link to="/rastrear" className="hover:opacity-80 transition">
            Rastrear Ônibus
          </Link>
          <Link to="/chegadas" className="hover:opacity-80 transition">
            Quadro de Chegadas
          </Link>
          <Link to="/denuncias" className="hover:opacity-80 transition">
            Denunciar Problema
          </Link>
          <Link to="/history" className="hover:opacity-80 transition">
            Histórico
          </Link>
          <Link to="/dashboard" className="hover:opacity-80 transition">
            CCO
          </Link>
        </div>

        <div className={`flex items-center gap-2 px-3 py-2 rounded-full ${healthColor}`}>
          {isStale ? (
            <>
              <AlertCircle className="w-4 h-4" />
              <span className="text-xs font-semibold">Offline</span>
            </>
          ) : health?.status === "ok" ? (
            <>
              <Check className="w-4 h-4" />
              <span className="text-xs font-semibold">Online</span>
            </>
          ) : (
            <>
              <AlertCircle className="w-4 h-4" />
              <span className="text-xs font-semibold">Degraded</span>
            </>
          )}
        </div>
      </div>
    </nav>
  );
};

export default Navigation;
