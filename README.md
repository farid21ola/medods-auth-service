# Medods Auth Service

Сервис аутентификации пользователей для Medods.

---

## Описание

REST API для генерации, обновления и инвалидирования access/refresh токенов пользователей. 
Документация доступна по адресу: `http://localhost:<SERVER_PORT>/swagger/index.html` после запуска сервиса.

---

## Быстрый старт (Docker Compose)

### 1. Настройте переменные окружения

Создайте файл `.env` в корне проекта и заполните его следующими переменными:

```env
# Для логирования (local/prod/dev)
ENV=local

# JWT
JWT_SECRET=your_jwt_secret_key

# Время жизни токенов
ACCESS_TTL=30m
REFRESH_TTL=720h

# Webhook (если используется)
WEBHOOK_URL=https://httpbin.org/anything
USER_AGENT=MedodsAuthService/1.0

# Сервер
SERVER_PORT=8081
TIMEOUT=10s
IDLE_TIMEOUT=60s

# База данных (PostgreSQL)
DB_HOST=db
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=admin
DB_NAME=auth-service

# Для миграции:
# 1 — только структура БД (без тестовых данных)
# 2 — структура БД + тестовые данные
MIGRATION_LEVEL=1
```

---

### 2. Запустите сервисы через Docker Compose

```sh
docker-compose -f docker-compose.yml up -d
```

- Сервис будет доступен на порту, указанном в `SERVER_PORT` (по умолчанию 8081).
- Swagger: [http://localhost:8081/swagger/index.html](http://localhost:8081/swagger/index.html)

---

### 4. Смена уровня миграции

- **Без тестовых данных:**
  ```env
  MIGRATION_LEVEL=1
  ```
- **С тестовыми данными:**
  ```env
  MIGRATION_LEVEL=2
  ```

После изменения MIGRATION_LEVEL выполните:
```sh
docker-compose up -d migrate
```
или перезапустите весь стек:
```sh
docker-compose up -d
```

---

### 5. Остановка сервисов

```sh
docker-compose down
```

---

## Полезные команды

- **Применить миграции до нужного уровня:**
  ```sh
  MIGRATION_LEVEL=2 docker-compose up -d migrate
  ```
- **Остановить и удалить все контейнеры, сети, тома:**
  ```sh
  docker-compose down -v
  ```

## Тестовые данные

- 1 Пользователь с guid
```
7cffbec9-676c-4a86-9384-273c3a88510a
```
- Тестовые данные нужны для тестирования сервиса, так как сам сервис не предполагает создания пользователя.
