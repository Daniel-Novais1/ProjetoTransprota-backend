import type { ReactNode } from "react";

// ============= TERMINAL =============
export interface Terminal {
  id: number;
  nome: string;
  lat: number;
  lng: number;
}

// ============= GPS & LOCALIZAÇÃO =============
export interface GPSData {
  bus_id: string;
  latitude: number;
  longitude: number;
  timestamp: string;
}

export interface StatusResponse {
  bus_id: string;
  status: "Em trânsito" | "No Terminal";
  terminal?: string;
  distancia_metros?: number;
}

// ============= ROTAS =============
export interface RouteStep {
  numero_linha: string;
  nome_linha: string;
  paradas: string[];
  tempo_total_minutos: number;
}

export interface RouteResponse {
  origem: string;
  destino: string;
  tipo: "direta" | "com_transferencia";
  steps: RouteStep[];
  cached: boolean;
}

export interface RouteRequest {
  origem: string;
  destino: string;
}

// ============= DENÚNCIAS =============
export enum TipoDenuncia {
  Lotado = "Lotado",
  Atrasado = "Atrasado",
  NaoParou = "Não Parou",
  ArEstragado = "Ar Estragado",
  Sujo = "Sujo",
}

export interface SubmeterDenunciaRequest {
  user_id: string;
  bus_line: string;
  bus_id: string;
  type: TipoDenuncia;
  latitude: number;
  longitude: number;
  evidence_url?: string;
}

export interface Denuncia {
  id: string;
  user_id: string;
  bus_line: string;
  bus_id: string;
  type: TipoDenuncia;
  location: string;
  timestamp: string;
  evidence_url?: string;
  trust_score: number;
}

export interface ListaDenunciasResponse {
  total: number;
  denuncias: Denuncia[];
}

export interface TrustScore {
  user_id: string;
  score: number;
  level: "Suspeito" | "Cidadão" | "Fiscal da Galera";
}

// ============= HEALTH =============
export interface HealthResponse {
  status: "ok" | "degraded" | "offline";
  timestamp: string;
  database: "ok" | "error" | "unknown";
  redis: "ok" | "error" | "unknown";
  uptime: number;
}

export interface ErrorResponse {
  error: string;
}

// ============= CHEGADAS =============
export interface ArrivalInfo {
  linha: string;
  proximoTerminal: string;
  etaMinutos: number;
  distanciaMetros: number;
  status: "Em trânsito" | "No Terminal";
}
