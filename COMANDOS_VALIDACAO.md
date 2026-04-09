# 🚀 Comandos para Criar e Validar Schema de Rotas

## PASSO 1: Copiar arquivo SQL para o container Docker

```powershell
docker cp "c:\programaçao\projetoTransprota\create_schema_rotas.sql" transprota_db:/schema.sql
```

**Resultado esperado:** Nenhuma mensagem de erro

---

## PASSO 2: Executar o arquivo SQL no PostgreSQL

```powershell
docker exec -e PGPASSWORD=password123 transprota_db psql -U admin -d transprota -f /schema.sql
```

**Resultado esperado:**
```
CREATE TABLE
CREATE INDEX
CREATE INDEX
CREATE TABLE
CREATE INDEX
CREATE INDEX
CREATE TABLE
CREATE INDEX
CREATE INDEX
CREATE INDEX
CREATE INDEX
INSERT 0 7
INSERT 0 3
INSERT 0 3
INSERT 0 3
INSERT 0 3
                status                
---------------------------------------
 ✅ Schema criado com sucesso!
(1 row)

 total_paradas 
---------------
            7
(1 row)

 total_linhas 
--------------
            3
(1 row)

 total_itinerarios 
-------------------
            9
(1 row)
```

---

## PASSO 3: Validar Estrutura das Tabelas

### 3.1 - Ver estrutura de pontos_parada

```powershell
docker exec -e PGPASSWORD=password123 transprota_db psql -U admin -d transprota -c "\d pontos_parada;"
```

**Resultado esperado:**
```
                 Table "public.pontos_parada"
   Column   |       Type       | Collation | Nullable |             Default              
-----------+------------------+-----------+----------+----------------------------------
 id        | integer          |           | not null | nextval('pontos_parada_id_seq'::)
 nome      | character varying |           | not null | 
 latitude  | numeric          |           | not null | 
 longitude | numeric          |           | not null | 
 tipo      | character varying |           |          | 'parada'::character varying
 criado_em | timestamp without time zone |   | | CURRENT_TIMESTAMP

Indexes:
    "pontos_parada_pkey" PRIMARY KEY, btree (id)
    "pontos_parada_nome_key" UNIQUE, btree (nome)
    "idx_pontos_parada_nome" btree (nome)
```

---

### 3.2 - Ver estrutura de linhas_onibus

```powershell
docker exec -e PGPASSWORD=password123 transprota_db psql -U admin -d transprota -c "\d linhas_onibus;"
```

**Resultado esperado:**
```
                Table "public.linhas_onibus"
     Column     |       Type       | Collation | ... | Default
----------------+------------------+-----------+-----+--------
 id             | integer          |           | ... | 
 numero_linha   | character varying |           | ... | 
 nome_linha     | character varying |           | ... | 
 descricao      | text             |           | ... | 
 status         | character varying |           | ... | 'ativa'
 empresa        | character varying |           | ... | 
 tipo_servico   | character varying |           | ... | 'regular'
 criado_em      | timestamp        |           | ... | CURRENT_TIMESTAMP

Indexes:
    "linhas_onibus_pkey" PRIMARY KEY, btree (id)
    "linhas_onibus_numero_linha_key" UNIQUE, btree (numero_linha)
    "idx_linhas_numero" btree (numero_linha)
    "idx_linhas_status" btree (status)
```

---

### 3.3 - Ver estrutura de itinerarios

```powershell
docker exec -e PGPASSWORD=password123 transprota_db psql -U admin -d transprota -c "\d itinerarios;"
```

**Resultado esperado:**
```
                  Table "public.itinerarios"
         Column         |       Type       | ... |  Default
------------------------+------------------+-----+----------
 id                     | integer          | ... | 
 linha_id               | integer          | ... | 
 parada_id              | integer          | ... | 
 ordem_parada           | integer          | ... | 
 tempo_estimado_anterior_minutos | integer | ... | 
 eh_ponto_integracao    | boolean          | ... | false
 criado_em              | timestamp        | ... | CURRENT_TIMESTAMP

Indexes:
    "itinerarios_pkey" PRIMARY KEY, btree (id)
    "itinerarios_linha_id_ordem_parada_key" UNIQUE, btree (linha_id, ordem_parada)
    "idx_itinerarios_linha" btree (linha_id)
    "idx_itinerarios_ordem" btree (linha_id, ordem_parada)
    "idx_itinerarios_parada" btree (parada_id)
    "idx_integracao" btree (eh_ponto_integracao) WHERE eh_ponto_integracao = true

Foreign-key constraints:
    "itinerarios_linha_id_fkey" FOREIGN KEY (linha_id) REFERENCES linhas_onibus(id) ON DELETE CASCADE
    "itinerarios_parada_id_fkey" FOREIGN KEY (parada_id) REFERENCES pontos_parada(id) ON DELETE RESTRICT
```

---

## PASSO 4: Validar Dados Inseridos

### 4.1 - Listar todas as paradas

```powershell
docker exec -e PGPASSWORD=password123 transprota_db psql -U admin -d transprota -c "SELECT id, nome, tipo, latitude, longitude FROM pontos_parada ORDER BY id;"
```

**Resultado esperado:**
```
 id |            nome            |    tipo    |  latitude  | longitude  
----+----------------------------+------------+------------+------------
  1 | Terminal Centro            | terminal   | -15.797500 | -47.891900
  2 | Eixo Anhanguera            | parada     | -15.780000 | -47.905000
  3 | Terminal Novo Mundo        | terminal   | -16.679900 | -49.213800
  4 | Terminal Padre Pelágio     | terminal   | -16.661700 | -49.324200
  5 | Terminal Praça da Bíblia   | terminal   | -16.673300 | -49.239400
  6 | Setor Comercial Sul        | parada     | -15.790000 | -47.880000
  7 | UFG Campus Samambaia       | parada     | -16.000000 | -48.950000
(7 rows)
```

---

### 4.2 - Listar todas as linhas

```powershell
docker exec -e PGPASSWORD=password123 transprota_db psql -U admin -d transprota -c "SELECT numero_linha, nome_linha, status, empresa FROM linhas_onibus ORDER BY numero_linha;"
```

**Resultado esperado:**
```
 numero_linha |     nome_linha      | status |  empresa
--------------+---------------------+--------+----------
 101          | Eixo Anhanguera     | ativa  | RMTC
 102          | Setor Comercial Sul | ativa  | RMTC
 103          | Integração Bíblia   | ativa  | RMTC
(3 rows)
```

---

### 4.3 - Ver rotas completas (Linha + Paradas em Ordem)

```powershell
docker exec -e PGPASSWORD=password123 transprota_db psql -U admin -d transprota -c "
SELECT 
    l.numero_linha,
    l.nome_linha,
    i.ordem_parada,
    p.nome as parada,
    i.tempo_estimado_anterior_minutos,
    i.eh_ponto_integracao
FROM linhas_onibus l
JOIN itinerarios i ON l.id = i.linha_id
JOIN pontos_parada p ON i.parada_id = p.id
ORDER BY l.numero_linha, i.ordem_parada;"
```

**Resultado esperado:**
```
 numero_linha |     nome_linha      | ordem_parada |            parada            | tempo_estimado_anterior_minutos | eh_ponto_integracao
--------------+---------------------+--------------+------------------------------+--------------------------------+-----------------
 101          | Eixo Anhanguera     |            1 | Terminal Centro              |                                | t
 101          | Eixo Anhanguera     |            2 | Eixo Anhanguera              |                               10 | f
 101          | Eixo Anhanguera     |            3 | Terminal Novo Mundo          |                               12 | t
 102          | Setor Comercial Sul |            1 | Terminal Centro              |                                | t
 102          | Setor Comercial Sul |            2 | Setor Comercial Sul          |                                8 | f
 102          | Setor Comercial Sul |            3 | UFG Campus Samambaia         |                               15 | f
 103          | Integração Bíblia   |            1 | Terminal Padre Pelágio       |                                | t
 103          | Integração Bíblia   |            2 | Terminal Praça da Bíblia     |                               10 | t
 103          | Integração Bíblia   |            3 | Terminal Centro              |                               12 | t
(9 rows)
```

---

### 4.4 - Ver Pontos de Integração (onde trocar de linha)

```powershell
docker exec -e PGPASSWORD=password123 transprota_db psql -U admin -d transprota -c "
SELECT 
    p.id,
    p.nome,
    COUNT(DISTINCT l.id) as total_linhas,
    STRING_AGG(l.numero_linha, ', ' ORDER BY l.numero_linha) as linhas
FROM pontos_parada p
JOIN itinerarios i ON p.id = i.parada_id
JOIN linhas_onibus l ON i.linha_id = l.id
WHERE i.eh_ponto_integracao = TRUE
GROUP BY p.id, p.nome
ORDER BY total_linhas DESC;"
```

**Resultado esperado:**
```
 id |           nome            | total_linhas | linhas
----+---------------------------+--------------+--------
  1 | Terminal Centro           |            2 | 101, 103
  4 | Terminal Padre Pelágio    |            1 | 103
  5 | Terminal Praça da Bíblia  |            1 | 103
  3 | Terminal Novo Mundo       |            1 | 101
(4 rows)
```

---

## ✅ Checklist de Validação

- [ ] Passo 1: Arquivo copiado para container
- [ ] Passo 2: SQL executado sem erros (CREATE TABLE, INSERT)
- [ ] Passo 3.1: Tabela `pontos_parada` existe com 5 colunas
- [ ] Passo 3.2: Tabela `linhas_onibus` existe com 7 colunas
- [ ] Passo 3.3: Tabela `itinerarios` existe com Foreign Keys
- [ ] Passo 4.1: 7 paradas inseridas
- [ ] Passo 4.2: 3 linhas inseridas
- [ ] Passo 4.3: 9 itinerários inseridos (3 por linha)
- [ ] Passo 4.4: Pontos de integração identificados corretamente

---

## 🎯 O Que Fazer Agora

### Próxima Integração: Conectar com main.go

Seu `main.go` atual usa a tabela `locations`. Você pode:

**Opção A:** Migrar para usar `pontos_parada` (recomendado)
- Atualizar queries de SELECT em `/terminais` 
- Atualizar cálculo de distância em `/gps/:id/status`
- Manter compatibilidade com Redis

**Opção B:** Sincronizar `locations` ↔ `pontos_parada`
- Manter main.go como está
- Usar view ou trigger para manter dois schemas em sync

**Qual você prefere?**
