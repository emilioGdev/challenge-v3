# Guia de Operação e Manutenção

**Projeto:** Solução de Telemetria de Frota (MVP)  
**Versão:** 1.0 (Arquitetura com Filas - Nível 5)

---

## 1. Pré-requisitos

Certifique-se de ter o seguinte software instalado:

- **Git:** Controle de versão  
- **Go:** Versão 1.24 ou superior  
- **Docker:** Containerização  
- **Docker Compose:** Orquestração dos containers

---

## 2. Configuração Inicial

### Clonar o repositório

```bash
git clone <URL_DO_SEU_REPOSITORIO>
cd challenge-v3
```

### Configurar variáveis de ambiente

Copie o arquivo de exemplo:

```bash
cp .env.example .env

cp .env.example .env
```

Edite o `.env` com suas credenciais AWS e, se necessário, altere dados do banco:

```env
DB_HOST=db
DB_PORT=5432
DB_USER=challengeuser
DB_PASSWORD=challengepassword
DB_NAME=telemetry_db

NATS_URL=nats://nats:4222

AWS_ACCESS_KEY_ID=SUA_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY=SUA_SECRET_ACCESS_KEY
AWS_REGION=us-east-1
REKOGNITION_COLLECTION_ID=fleet_drivers
```

### Instalar dependências Go

```bash
go mod tidy
```

---

## 3. Execução do Sistema

### Iniciar todos os serviços

```bash
docker-compose up --build -d
```

### Verificar status dos containers

```bash
docker-compose ps
```

### Visualizar logs

```bash
docker-compose logs -f

docker-compose logs -f worker
```

### Parar os serviços

```bash
docker-compose down
```

### Resetar com remoção de volumes

```bash
docker-compose down -v
```

---

## 4. Executar Testes Automatizados

Garanta que `DB_HOST=db` esteja no `.env`. Execute:

```bash
docker-compose run --rm app go test ./...
```

---

## 5. Manutenção e Verificação

### Acessar o banco de dados

```bash
docker-compose exec db psql -U challengeuser -d telemetry_db
```

No shell `psql`, use:

- `\dt` — listar tabelas  
- `SELECT * FROM photo;` — consultar dados  
- `\q` — sair

### Monitorar NATS

Acesse o painel web do NATS:  
[http://localhost:8222/](http://localhost:8222/)

---

## 6. Atualização da Aplicação

```bash
git pull origin main

docker-compose down

docker-compose up --build -d
```

---

Este guia cobre a operação completa da aplicação em ambiente de desenvolvimento.