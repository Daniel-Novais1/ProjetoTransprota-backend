# Frontend Setup - Visualização em Tempo Real

## 📦 Dependências Adicionadas

As seguintes dependências foram adicionadas ao `package.json`:
- `react-toastify@^9.1.3` - Sistema de notificações
- `socket.io-client@^4.7.2` - Cliente WebSocket

## 🚀 Passos para Executar

### 1. Instalar Dependências
```bash
cd frontend
npm install
```

### 2. Configurar Variáveis de Ambiente
Crie ou atualize o arquivo `.env` no diretório `frontend/`:
```env
VITE_API_URL=http://localhost:8081
VITE_WS_URL=ws://localhost:8081/ws
```

### 3. Iniciar Servidor de Desenvolvimento
```bash
npm run dev
```

O frontend estará disponível em `http://localhost:5173`

## 🎯 Novos Componentes Criados

### 1. `src/hooks/useRealtime.ts`
Hook personalizado para conexão WebSocket com o backend.
- Conecta ao endpoint `/ws`
- Escuta canais `bus_updates` e `bi_alerts`
- Gerencia reconexão automática
- Retorna dados em tempo real de ônibus e alertas

### 2. `src/components/RealtimeBusMap.tsx`
Componente de mapa interativo com react-leaflet.
- Plota marcadores de ônibus em tempo real
- Ícone customizado para ônibus
- Popup com informações detalhadas (velocidade, coordenadas, timestamp)
- Auto-fit bounds para mostrar todos os ônibus
- Indicador de status de conexão WebSocket

### 3. `src/components/AlertToasts.tsx`
Sistema de notificações com react-toastify.
- Alertas de engarrafamento (BI Analytics)
- Alertas de geofencing (saída de rota)
- Toasts com cores e estilos customizados
- Auto-dismiss configurável

### 4. `src/components/Login.tsx`
Página de login para autenticação JWT.
- Formulário de login com username/password
- Armazena token JWT no localStorage
- Credenciais padrão: admin / admin123
- Redirecionamento após login

## 🔐 Autenticação

### Fluxo de Login
1. Usuário acessa `/` → redirecionado para Login
2. Usuário faz login → token JWT armazenado no localStorage
3. Todas as requisições subsequentes enviam `Authorization: Bearer <token>`
4. Token 401 → automaticamente removido e redirecionado para Login

### API Client Atualizado
O `src/api/client.ts` foi atualizado para:
- Adicionar JWT token do localStorage em todas as requisições
- Interceptor para capturar erros 401 e limpar token
- Remoção de `X-API-Key` (agora usa JWT)

## 📡 WebSocket Integration

### Backend Endpoint
- **URL:** `ws://localhost:8081/ws`
- **Protocolo:** WebSocket nativo (gorilla/websocket)

### Canais Escutados
1. **bus_updates** - Atualizações de posição de ônibus
   ```json
   {
     "device_id": "bus-001",
     "lat": -16.6869,
     "lng": -49.2648,
     "speed": 45.5,
     "timestamp": "2024-01-01T12:00:00Z"
   }
   ```

2. **bi_alerts** - Alertas de analytics (engarrafamento)
   ```json
   {
     "alert": "CONGESTION_DETECTED",
     "avg_speed": 32.5,
     "baseline": 40.0,
     "reduction": 18.75,
     "bus_count": 50,
     "timestamp": "2024-01-01T12:00:00Z"
   }
   ```

3. **geofence_alerts** - Alertas de geofencing (saída de rota)
   ```json
   {
     "device_id": "bus-001",
     "lat": -16.6869,
     "lng": -49.2648,
     "alert": "GEOFENCE_BREACH",
     "fence": "Cerca de Goiânia",
     "timestamp": "2024-01-01T12:00:00Z"
   }
   ```

## 🗺️ Mapa Interativo

### React Leaflet
O componente usa `react-leaflet` para renderizar o mapa:
- **Provider:** OpenStreetMap
- **Center:** Goiânia (-16.6869, -49.2648)
- **Zoom:** 13 (ajustável)
- **Markers:** Ícones customizados de ônibus
- **Auto-fit:** Ajusta automaticamente para mostrar todos os ônibus

### Estilo do Mapa
- Altura: 100% do container
- Border-radius: 8px
- Shadow-lg para destaque
- Indicador de conexão no canto superior esquerdo

## 🚨 Sistema de Alertas

### Toast Notifications
- **Engarrafamento (BI):** Toast vermelho no canto superior direito
- **Geofencing:** Toast laranja no canto superior esquerdo
- **Auto-dismiss:** 10-15 segundos
- **Draggable:** Sim
- **Pause on Hover:** Sim

### Estilo dos Toasts
- **BI Alert:** Background vermelho claro, border-left vermelho
- **Geofence Alert:** Background laranja claro, border-left laranja

## 🔄 Integração no App.tsx

### Novas Rotas
- `/` - Dashboard com mapa em tempo real (autenticado)
- `/realtime` - Mapa em tela cheia (autenticado)
- `/` (não autenticado) - Página de login

### Componentes Integrados
- `Login` - Renderizado quando não autenticado
- `AlertToasts` - Renderizado globalmente para notificações
- `RealtimeBusMap` - Integrado no dashboard
- Botão de logout no canto superior direito

## 📊 Estrutura de Dados

### RealtimeData Interface
```typescript
interface RealtimeData {
  buses: Map<string, BusUpdate>;
  biAlert: BIAlert | null;
  geofenceAlert: GeofenceAlert | null;
}
```

### BusUpdate
```typescript
interface BusUpdate {
  device_id: string;
  lat: number;
  lng: number;
  speed: number;
  timestamp: string;
}
```

## 🎨 Estilo e UI

### TailwindCSS
O frontend usa TailwindCSS para estilização:
- Grid layout responsivo
- Cards com shadow-lg e rounded-lg
- Cores semânticas (vermelho para erros, verde para sucesso)
- Transições suaves em hover

### Lucide Icons
Ícones da biblioteca lucide-react:
- `LogOut` - Botão de logout
- `AlertCircle` - Indicador de erro
- `User` - Ícone de usuário no login
- `Lock` - Ícone de senha no login

## 🐛 Troubleshooting

### Erro: Cannot find module 'react'
**Solução:** Execute `npm install` no diretório `frontend/`

### Erro: WebSocket connection failed
**Solução:** Verifique se o backend está rodando em `http://localhost:8081`

### Erro: 401 Unauthorized
**Solução:** Faça login novamente para obter um novo token JWT

### Erro: Map markers not showing
**Solução:** Verifique se o WebSocket está recebendo dados de `bus_updates`

## 📝 Próximos Passos

1. Executar `npm install` no diretório `frontend/`
2. Configurar variáveis de ambiente em `.env`
3. Iniciar backend: `.\transprota.exe`
4. Iniciar frontend: `npm run dev`
5. Acessar `http://localhost:5173`
6. Fazer login com `admin / admin123`
7. Ver o "pulso" do sistema acontecendo visualmente!

---

*Frontend setup para visualização em tempo real - TranspRota Squad*
