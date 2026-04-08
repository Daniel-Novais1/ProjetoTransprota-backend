import React from 'react';
import { Link } from 'react-router-dom';
import { MapPin, Zap, AlertCircle } from 'lucide-react';

function Navigation({ health }) {
  const statusColor =
    health?.status === 'ok'
      ? 'bg-green-100 text-green-800'
      : health?.status === 'degraded'
      ? 'bg-yellow-100 text-yellow-800'
      : 'bg-red-100 text-red-800';

  return (
    <nav className="bg-white shadow-md sticky top-0 z-50">
      <div className="container mx-auto px-4 py-4 flex justify-between items-center">
        <Link to="/" className="flex items-center gap-2 font-bold text-2xl text-indigo-600 hover:text-indigo-700">
          <Zap className="w-6 h-6" />
          TranspRota
        </Link>

        <div className="flex gap-6 items-center">
          <Link
            to="/"
            className="flex items-center gap-2 text-gray-700 hover:text-indigo-600 transition"
          >
            <MapPin className="w-5 h-5" />
            Planejar Rota
          </Link>

          <Link
            to="/rastrear"
            className="flex items-center gap-2 text-gray-700 hover:text-indigo-600 transition"
          >
            <Zap className="w-5 h-5" />
            Rastrear Ônibus
          </Link>

          <Link
            to="/denuncias"
            className="flex items-center gap-2 text-gray-700 hover:text-indigo-600 transition"
          >
            <AlertCircle className="w-5 h-5" />
            Denúncias
          </Link>

          <div className={`px-3 py-1 rounded-full text-sm font-semibold ${statusColor}`}>
            {health?.status === 'ok' ? '✓ Online' : `⚠ ${health?.status}`}
          </div>
        </div>
      </div>
    </nav>
  );
}

export default Navigation;
