# Procedimentos de Backup e Recuperação

**Projeto:** Solução de Telemetria de Frota (MVP)  
**Versão:** 1.0  

---

## 1. Visão Geral

Este documento descreve os procedimentos padrão para realizar o backup e a recuperação de dados da aplicação. O objetivo é garantir a integridade dos dados, a continuidade do negócio e minimizar o tempo de inatividade (Downtime) e a perda de dados (RPO - Recovery Point Objective) em caso de um incidente.

---

## 2. Ativos Críticos a Serem Backupeados

### Banco de Dados PostgreSQL

- **Descrição:** Contém todos os dados históricos de telemetria (giroscópio, GPS, fotos) e os resultados da análise de reconhecimento.  
- **Componente:** Container Docker `challenge_db_postgres`  
- **Estratégia:** Backup lógico diário utilizando `pg_dump`

### Coleção AWS Rekognition

- **Descrição:** Contém os dados biométricos (Face IDs) dos rostos indexados  
- **Estratégia:** Serviço gerenciado com alta durabilidade. Backup manual não é necessário. Em caso de exclusão acidental, a recuperação está descrita na seção 5.

---

## 3. Backup do PostgreSQL

A estratégia consiste em criar um dump lógico do banco (`.sql`), contendo comandos para recriar o esquema e os dados.

- **Ferramenta:** `pg_dump`  
- **Frequência:** Diária

### Comando

Executar na máquina host:

```bash
docker-compose exec -T db pg_dump -U challengeuser telemetry_db > backup_$(date +%Y-%m-%d_%H-%M-%S).sql
```

- `exec -T`: necessário para redirecionar corretamente  
- `db`: nome do serviço no `docker-compose.yml`  
- `> backup_...sql`: cria o arquivo de backup localmente

### Armazenamento Seguro

- **Destinos sugeridos:** AWS S3, Google Cloud Storage, servidor de backup  
- **Política de retenção:** Ex: diários (7 dias), semanais (4 semanas)

---

## 4. Restauração do PostgreSQL

### Ferramenta

- `psql` (cliente do PostgreSQL)

### Passos

1. **Preparar ambiente**

Garanta que um container limpo esteja ativo:

```bash
docker-compose down -v
docker-compose up -d db
```

2. **Executar restore**

```bash
cat backup_2025-06-26_12-30-00.sql | docker-compose exec -T db psql -U challengeuser -d telemetry_db
```

3. **Verificar restauração**

Conecte-se ao banco e use SQL:

```sql
SELECT COUNT(*) FROM photo;
SELECT * FROM gps LIMIT 10;
```

---

## 5. Considerações Adicionais

### Rekognition: Recuperação da Coleção

Caso a coleção de rostos seja apagada:

- Leia a tabela `photo` do banco restaurado  
- Use a API `IndexFaces` do Rekognition para reenviar as imagens  
- Assim, a coleção pode ser repopulada automaticamente via script