import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "./index.css";

const rootElement = document.getElementById("root");

if (!rootElement) {
  console.error("❌ ERRO CRÍTICO: Elemento #root não encontrado no DOM!");
} else {
  const root = ReactDOM.createRoot(rootElement);
  root.render(<App />);
  console.log("🚀 V1.2 ESTÁVEL - ISOLAMENTO CONFIRMADO");
}
