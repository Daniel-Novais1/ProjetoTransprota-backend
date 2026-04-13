import React from "react";
import { createHashRouter, RouterProvider } from "react-router-dom";
import { ComponenteSaaS } from "@/components/ComponenteSaaS";
import { ProtectedRoute } from "@/components/ProtectedRoute";
import LOGISTICA_FANTASMA from "@/components/LOGISTICA_FANTASMA";

const routes = [
  { path: "/", element: <LOGISTICA_FANTASMA /> },
  { path: "/dashboard", element: <ComponenteSaaS /> },
  { 
    path: "/settings", 
    element: (
      <ProtectedRoute>
        <div className="p-8">
          <h1 className="text-2xl font-bold text-gray-800">Configurações</h1>
          <p className="text-gray-600 mt-2">Área protegida - requer autenticação</p>
        </div>
      </ProtectedRoute>
    )
  },
  {
    path: "*",
    element: (
      <div className="min-h-screen flex items-center justify-center bg-gray-100">
        <div className="text-center">
          <h1 className="text-4xl font-bold text-gray-800 mb-4">Página Não Encontrada</h1>
          <p className="text-gray-600 mb-6">A rota solicitada não existe</p>
          <div className="space-x-4">
            <a href="#/" className="inline-block px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700">
              Ir para Home
            </a>
            <a href="#/dashboard" className="inline-block px-6 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700">
              Ir para Dashboard
            </a>
          </div>
        </div>
      </div>
    ),
  },
];

const router = createHashRouter(routes) as any;

export default function App() {
  return <RouterProvider router={router} />;
}
