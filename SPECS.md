 Especificações Técnicas - TranspRota

Este documento detalha as regras de negócio e a lógica do backend para o sistema de monitoramento de transporte público de Goiânia.

1. Sistema de Confiança (Trust Score)

O coração da fiscalização colaborativa. Cada usuário possui um TrustScore que determina o impacto da sua denúncia no sistema.

Regras de Pontuação:

Pontuação Inicial: 50 pontos (Neutro).

Denúncia Confirmada: +5 pontos (quando 3 ou mais usuários relatam o mesmo problema no mesmo veículo em um intervalo de 15 min).

Denúncia Falsa/Spam: -15 pontos (identificado por divergência de GPS ou múltiplos reports contraditórios).

Dossiê Completo: +10 pontos (denúncias com foto e localização em tempo real).

Níveis de Impacto:

0-20 (Suspeito): Denúncias são registradas mas não aparecem no mapa para outros usuários.

21-80 (Cidadão): Denúncias aparecem com um ícone padrão.

81-100 (Fiscal da Galera): Denúncias aparecem com destaque "Verificado pela Comunidade" e têm prioridade de notificação.

2. Coleta de Dados (GTFS & GPS)

O sistema deve processar dois fluxos de dados simultâneos:

Dados Estáticos (GTFS): Importados mensalmente da RMTC (horários planejados, nomes de paradas e trajetos).

Dados em Tempo Real: Coordenadas enviadas pelos usuários que estão "Dando Carona no GPS" dentro do ônibus.

3. Arquitetura de Dados (Go + PostGIS)

Estrutura da Denúncia (Model):

ID: UUID

UserID: UUID

BusLine: String (ex: "001", "048")

BusID: String (Prefixo do veículo)

Type: Enum (Lotado, Atrasado, Não Parou, Ar Estragado, Sujo)

Location: Geometry(Point, 4326)

Timestamp: DateTime

EvidenceURL: String (Link para imagem no storage)

4. Estratégia de Cache (Redis)

Para manter o desempenho no seu i3-8100, usaremos Redis para:

Localização dos Ônibus: Dados expiram a cada 30 segundos.

Sessões de Usuário: Armazenar o token de autenticação.

Hotspots de Denúncia: Cache das áreas com mais problemas na última 1 hora.

Documento em constante atualização conforme o desenvolvimento avança.
