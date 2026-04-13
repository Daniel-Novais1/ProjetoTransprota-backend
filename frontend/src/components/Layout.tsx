import React, { ReactNode } from "react";
import { Outlet } from "react-router-dom";
import Navigation from "./Navigation";

interface LayoutProps {
  children?: ReactNode;
}

export default function Layout({ children }: LayoutProps) {
  return (
    <div className="min-h-screen bg-gray-50 flex flex-col">
      <Navigation />
      <main className="flex-1 py-8">
        {children || <Outlet />}
      </main>
      <footer className="bg-gray-800 text-gray-300 text-center py-4 text-sm">
        TranspRota v1.1 | Precisão em Tempo Real | Monitoramento Ativo
      </footer>
    </div>
  );
}
