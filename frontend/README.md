# TranspRota Frontend v1.1

Frontend React para o sistema de monitoramento de ônibus em tempo real com planejamento de rotas e dashboard de compliance B2B.

## 📌 Versão Atual: v1.1

**Novidades nesta versão:**
- ✅ HashRouter implementado para compatibilidade
- ✅ Dashboard de Compliance B2B com Tailwind
- ✅ Sidebar persistente com navegação
- ✅ Grid de 3 Cards (Status, Alertas, Auditorias)
- ✅ Mock API Service para desenvolvimento
- ✅ ProtectedRoute para rotas autenticadas
- ✅ Security audit realizado
- ✅ Build de produção 100% verde

## 🎯 Funcionalidades

- **Planejador de Rotas**: Calcula a melhor rota entre dois pontos
- **Rastreador de Ônibus**: Monitora a localização em tempo real
- **Sistema de Denúncias**: Colaboração da comunidade para reportar problemas
- **Dashboard de Compliance**: Interface B2B para auditoria e telemetria
- **Status de Conformidade**: Monitoramento em tempo real da frota
- **Alertas de Telemetria**: Notificações de anomalias
- **Histórico de Auditorias**: Logs de ações administrativas
- **Trust Score**: Sistema de reputação para usuários

## 📦 Instalação

```bash
# Instalar dependências
npm install

# Criar arquivo .env baseado em .env.example
cp .env.example .env

# Editar .env com as configurações corretas
```

## 🚀 Desenvolvimento

```bash
# Usar o script de bypass de PATH (Windows)
start_dev.bat

# Ou manualmente
npm run dev
```

A aplicação estará disponível em `http://localhost:5173`

## 🏗️ Build para Produção

```bash
# Usar o script de build (Windows)
build.bat

# Ou manualmente
npm run build

# Preview do build de produção
npm run preview
```

## 📝 Variáveis de Ambiente

```env
VITE_API_URL=http://localhost:8080  # URL base da API
VITE_WS_URL=ws://localhost:8081/ws # WebSocket para atualizações em tempo real
```

## 🎨 Estrutura de Componentes

```
frontend/
├── src/
│   ├── components/
│   │   ├── DashboardPrincipal.tsx  # Dashboard de Compliance B2B
│   │   ├── ProtectedRoute.tsx       # Barreira de autenticação
│   │   ├── RouteMap.tsx            # Planejador de rotas (Home)
│   │   ├── Login.tsx               # Autenticação
│   │   └── ...                     # Outros componentes
│   ├── services/
│   │   └── api.ts                  # Mock API Service (desenvolvimento)
│   ├── api/
│   │   └── client.js               # Cliente Axios configurado
│   ├── App.tsx                     # Componente raiz com HashRouter
│   ├── main.tsx                    # Ponto de entrada
│   └── index.css                   # Tailwind CSS
├── public/
│   └── index.html
├── vite.config.js                  # Configuração Vite (porta 5173, host 0.0.0.0)
├── build.bat                       # Script de build para Windows
├── start_dev.bat                   # Script de desenvolvimento para Windows
├── tailwind.config.js
├── package.json
└── .env.example
```

## 🔌 Integração com API

A API é acessada via cliente Axios configurado em `src/api/client.js`:

```javascript
import { api } from './api/client'

// Calcular rota
const route = await api.calcularRota('Vila Pedroso', 'UFG')

// Rastrear ônibus
const location = await api.getBusLocation('BUS-001')

// Submeter denúncia
await api.submitReport(denunciaData)
```

## � Roteamento

O sistema usa **HashRouter** para compatibilidade:

- `#/` - Home (Planejador de Rotas)
- `#/dashboard` - Dashboard de Compliance
- `#/settings` - Configurações (protegido)
- `#/*` - Página não encontrada

## 🔐 Segurança

- JWT tokens armazenados no localStorage
- Rotas protegidas via `ProtectedRoute` component
- LocalStorage usado para armazenar ID do usuário
- HTTPS recomendado em produção
- Credenciais não expostas no código frontend

## 📱 Responsividade

A aplicação é totalmente responsiva usando Tailwind CSS, otimizada para:
- 📱 Celulares (320px+)
- 📱 Tablets (768px+)
- 💻 Desktops (1024px+)

## 🛠️ Tecnologias

- **React 18**: Framework UI
- **React Router v6.8.1**: Navegação com HashRouter
- **Axios**: Cliente HTTP
- **Vite 4.5.14**: Build tool
- **Tailwind CSS**: Estilos
- **Lucide React**: Ícones
- **TypeScript**: Tipagem estática

## 📄 Licença

MIT
