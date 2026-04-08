import axios, { AxiosInstance, AxiosError } from "axios";
import {
  RouteResponse,
  RouteRequest,
  GPSData,
  StatusResponse,
  Denuncia,
  ListaDenunciasResponse,
  SubmeterDenunciaRequest,
  HealthResponse,
  ErrorResponse,
} from "@/types";

class ApiClient {
  private client: AxiosInstance;

  constructor() {
    const apiUrl = import.meta.env.VITE_API_URL || "http://localhost:8080";
    const apiKey = import.meta.env.VITE_API_KEY || "default-key";

    this.client = axios.create({
      baseURL: apiUrl,
      timeout: 10000,
      headers: {
        "Content-Type": "application/json",
        "X-API-Key": apiKey,
      },
    });
  }

  async getHealth(): Promise<HealthResponse> {
    const { data } = await this.client.get<HealthResponse>("/health");
    return data;
  }

  async calcularRota(req: RouteRequest): Promise<RouteResponse> {
    const params = new URLSearchParams({
      origem: req.origem,
      destino: req.destino,
    });
    const { data } = await this.client.get<RouteResponse>(
      `/planejar?${params}`
    );
    return data;
  }

  async getBusLocation(busId: string): Promise<GPSData> {
    const { data } = await this.client.get<GPSData>(`/gps/${busId}`);
    return data;
  }

  async getBusStatus(busId: string): Promise<StatusResponse> {
    const { data } = await this.client.get<StatusResponse>(
      `/gps/${busId}/status`
    );
    return data;
  }

  async submitReport(
    report: SubmeterDenunciaRequest
  ): Promise<{ status: string; trust_score: number }> {
    const response = await this.client.post<{
      status: string;
      trust_score: number;
    }>("/denuncias", report);
    return response.data;
  }

  async getNearbyReports(
    latitude: number,
    longitude: number,
    radiusMeters: number = 1000
  ): Promise<Denuncia[]> {
    const params = new URLSearchParams({
      lat: latitude.toString(),
      lon: longitude.toString(),
      radius: radiusMeters.toString(),
    });
    const { data } = await this.client.get<ListaDenunciasResponse>(
      `/denuncias?${params}`
    );
    return data.denuncias;
  }

  static extractErrorMessage(error: AxiosError<ErrorResponse>): string {
    if (error.response?.data?.error) {
      return error.response.data.error;
    }
    return error.message || "Erro desconhecido";
  }
}

export const api = new ApiClient();
export default api;
