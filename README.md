## Log Aggregator

Микросервис на Go, который принимает файлы-логи, парсит их и агрегирует информацию, сохраняет полученные данные в PostgreSQL и предоставляет REST‑API для получения информации о сущностях.

## Функциональность

| Эндпоинт | Метод | Описание |
|----------|-------|----------|
| `/api/v1/parse/` | POST | Загружает и парсит ZIP-архив с логами |
| `/api/v1/topology/{log_id}` | GET | Возвращает топологию сети (узлы и порты) |
| `/api/v1/node/{node_id}` | GET | Детальная информация об узле |
| `/api/v1/port/{node_id}` | GET | Список портов узла |
| `/api/v1/log/{log_id}` | GET | Метаинформация о загруженном логе |

## Запуск

```bash
docker-compose up -d --build
```

## Примеры curl

### Загрузить и распарсить лог

```bash
curl -X POST http://localhost:8080/api/v1/parse/ \
  -H "Content-Type: application/json" \
  -d '{"file_path":"/data/log.zip"}'
```
### Получить топологию

```bash
curl http://localhost:8080/api/v1/topology/1   
```

### Получить детали узла

```bash
curl http://localhost:8080/api/v1/node/1 
```
### Получить порты узла

```bash
curl http://localhost:8080/api/v1/port/1 
```
### Получить мета-информацию о логе

```bash
curl http://localhost:8080/api/v1/log/1 
```