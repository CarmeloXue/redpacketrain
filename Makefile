DC=docker compose

.PHONY: start stop restart api-start api-stop api-restart consumer-start consumer-stop consumer-restart

start:
	$(DC) up -d --build

stop:
	$(DC) down

restart: stop start

api-start:
	$(DC) up -d api

api-stop:
	$(DC) stop api

api-restart: api-stop api-start

consumer-start:
	$(DC) up -d consumer

consumer-stop:
	$(DC) stop consumer

consumer-restart: consumer-stop consumer-start
