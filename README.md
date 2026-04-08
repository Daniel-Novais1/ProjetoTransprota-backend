# 🚌 TranspRota Goiânia

**O monitoramento inteligente do transporte coletivo na palma da mão.**

O TranspRota é um sistema de monitoramento e fiscalização colaborativa desenvolvido para resolver o problema da imprecisão de horários e falta de transparência no transporte público de Goiânia (RMTC).

---

## 🚀 Visão do Projeto
Diferente de apps que mostram apenas o horário teórico, o TranspRota utiliza uma lógica de **Trust Score** (sistema de pesos) para validar denúncias de usuários em tempo real, gerando dados confiáveis sobre lotação, atrasos e condições dos veículos.

## 🛠️ Stack Tecnológica
- **Linguagem:** [Go (Golang)](https://go.dev/) - Escolhida pela alta performance e concorrência.
- **Banco de Dados:** [PostgreSQL](https://www.postgresql.org/) com extensão **PostGIS** para inteligência geoespacial.
- **Cache:** [Redis](https://redis.io/) para processamento de localização em tempo real.
- **Infra:** Docker & Docker Compose para um ambiente isolado e escalável.

## 📋 Funcionalidades Planejadas
- [ ] **Mapa em Tempo Real:** Visualização de ônibus baseada em dados de GPS colaborativo.
- [ ] **Sistema de Denúncias:** Interface rápida para reportar lotação, ar-condicionado estragado ou atrasos.
- [ ] **Ranking de Confiança:** Usuários que colaboram com dados reais ganham maior peso no sistema.
- [ ] **Painel B2B/BI:** Geração de relatórios de eficiência para análise de frotas e terminais.

## ⚙️ Como rodar o projeto (Em breve)
Este projeto está em fase inicial de desenvolvimento (Estudante de Engenharia de Software - UFG).

```bash
# Clone o repositório
git clone [https://github.com/seu-usuario/transprota-backend.git](https://github.com/seu-usuario/transprota-backend.git)

# Suba a infraestrutura
docker-compose up -d
