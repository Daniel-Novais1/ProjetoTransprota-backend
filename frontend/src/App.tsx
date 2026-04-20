import React from "react";
import { HashRouter, Routes, Route, Navigate } from 'react-router-dom';
import Home from "./pages/Home";

export default function App() {
  return (
    <HashRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/dashboard" element={<Home />} />
        <Route path="*" element={<Home />} />
      </Routes>
    </HashRouter>
  );
}
