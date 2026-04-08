# TranspRota Frontend

Frontend React para o sistema de monitoramento de ônibus em tempo real com planejamento de rotas.

## 🎯 Funcionalidades

- **Planejador de Rotas**: Calcula a melhor rota entre dois pontos
- **Rastreador de Ônibus**: Monitora a localização em tempo real
- **Sistema de Denúncias**: Colaboração da comunidade para reportar problemas
- **Trust Score**: Sistema de reputação para usuários
- **Status da API**: Verificação de saúde das dependências

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
npm run dev
```

A aplicação estará disponível em `http://localhost:5173`

## 🏗️ Build para Produção

```bash
npm run build
npm run preview
```

## 📝 Variáveis de Ambiente

```env
VITE_API_URL=http://localhost:8080  # URL base da API
VITE_API_KEY=seu-api-key            # Chave de API para endpoints protegidos
```

## 🎨 Estrutura de Componentes

```
frontend/
├── src/
│   ├── components/
│   │   ├── Navigation.jsx          # Navegação principal
│   │   ├── RouteCalculator.jsx     # Planejador de rotas
│   │   ├── BusTracker.jsx          # Rastreador de ônibus
│   │   └── Reports.jsx             # Sistema de denúncias
│   ├── api/
│   │   └── client.js               # Cliente Axios configurado
│   ├── App.jsx                     # Componente raiz
│   ├── App.css                     # Estilos globais
│   ├── index.css                   # Tailwind CSS
│   └── main.jsx                    # Ponto de entrada
├── public/
│   └── index.html
├── vite.config.js
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

## 📱 Responsividade

A aplicação é totalmente responsiva usando Tailwind CSS, otimizada para:
- 📱 Celulares (320px+)
- 📱 Tablets (768px+)
- 💻 Desktops (1024px+)

## 🔐 Segurança

- Chave de API é enviada no header `X-API-Key`
- LocalStorage usado para armazenar ID do usuário
- HTTPS recomendado em produção

## 🛠️ Tecnologias

- **React 18**: Framework UI
- **React Router**: Navegação
- **Axios**: Cliente HTTP
- **Vite**: Build tool
- **Tailwind CSS**: Estilos
- **Lucide React**: Ícones

## 📄 Licença

MIT
