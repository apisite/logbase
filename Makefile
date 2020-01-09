# application project makefile

SHELL          = /bin/bash
CFG           ?= .env
PRG           ?= $(shell basename $$PWD)

# -----------------------------------------------------------------------------
# Build config

GO            ?= go
VERSION       ?= $(shell git describe --tags --always)
SOURCES       ?= cmd/*/*.go *.go

# -----------------------------------------------------------------------------
# Runtime data

APP_LISTEN    ?= :7070

PGHOST        ?= db
PGPORT        ?= 5432
PGDATABASE    ?= $(PRG)
PGUSER        ?= $(PRG)
PGPASSWORD    ?= $(shell < /dev/urandom tr -dc A-Za-z0-9 | head -c14; echo)
PGAPPNAME     ?= $(PRG)

# -----------------------------------------------------------------------------
# docker part

APP_IMAGE  ?= $(PRG)

# image prefix
PROJECT_NAME ?= $(PRG)

# docker-compose image
DC_VER ?= 1.23.2

# -----------------------------------------------------------------------------
# dcape part

# dcape containers name prefix
DCAPE_PROJECT_NAME ?= dcape
# dcape postgresql container name
DCAPE_DB           ?= $(PRG)_db_1


define CONFIG_DEFAULT
# ------------------------------------------------------------------------------
# application config file, generated by make $(CFG)

# App/Docker listen addr
APP_LISTEN=$(APP_LISTEN)

# Docker image tag
APP_IMAGE=$(APP_IMAGE)

# Database

# Host
PGHOST=$(PGHOST)
# Port
PGPORT=$(PGPORT)
# Name
PGDATABASE=$(PGDATABASE)
# User
PGUSER=$(PGUSER)
# Password
PGPASSWORD=$(PGPASSWORD)
# App name
PGAPPNAME=$(PGAPPNAME)

endef
export CONFIG_DEFAULT

# ------------------------------------------------------------------------------

-include $(CFG)
export

.PHONY: all api dep build run lint test up up-db down psql clean help

all: help

build: dep ## Build the binary file for server
	@go build -i -v -ldflags "-X main.version=`git describe --tags`" ./cmd/$(PRG)/

## Build app used in docker from scratch
#build-standalone: cov vet lint lint-more

build-standalone:
	CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.version=`git describe --tags`" -a ./cmd/$(PRG)/

run: ## Build and run binary
	$(GO) run -ldflags "-X main.version=$(VERSION)" ./cmd/$(PRG)/ --listen ${APP_LISTEN}

lint: ## Run linter
	@golangci-lint run ./...

# ------------------------------------------------------------------------------
# Docker

start-hook: dcape-db-create up

up: ## Start app container
up: CMD=up -d app
up: dc

up-db: ## Start pg container only
up-db: CMD=up -d db
up-db: dc

up-all: ## Start pg & app containers
up-all: CMD=up -d
up-all: dc

down: ## Stop containers and remove them
down: CMD=rm -f -s
down: dc

# $$PWD используется для того, чтобы текущий каталог был доступен 
# в контейнере docker-compose по тому же пути
# и относительные тома новых контейнеров могли его использовать
dc: docker-compose.yml ## Run docker-compose (make dc CMD=build)
	@docker run --rm  \
	  -v /var/run/docker.sock:/var/run/docker.sock \
	  -v $$PWD:$$PWD \
	  -w $$PWD \
	  docker/compose:$(DC_VER) \
	  -p $$PROJECT_NAME \
	  $(CMD)

# ------------------------------------------------------------------------------
# DB operations with docker and [dcape](https://github.com/dopos/dcape)

# (internal) Wait for postgresql container start
docker-wait:
	@echo -n "Checking PG is ready..."
	@until [[ `docker inspect -f "{{.State.Health.Status}}" $$DCAPE_DB` == healthy ]] ; do sleep 1 ; echo -n "." ; done
	@echo "Ok"

dcape-db-create: docker-wait ## Create user, db and load dump
	@echo "*** $@ ***" ; \
	docker exec -i $$DCAPE_DB psql -U postgres -c "CREATE USER \"$$PGUSER\" WITH PASSWORD '$$PGPASSWORD';" 2> >(grep -v "already exists" >&2) || true ; \
	docker exec -i $$DCAPE_DB psql -U postgres -c "CREATE DATABASE \"$$PGDATABASE\" OWNER \"$$PGUSER\";" 2> >(grep -v "already exists" >&2) || db_exists=1 ; \
	if [[ ! "$$db_exists" ]] ; then \
	    for f in sql/*.sql ; do cat $$f ; done | docker exec -i $$DCAPE_DB psql -U "$$PGUSER" -1 -X ; \
	    echo "Restore completed" ; \
	fi

dcape-db-drop: docker-wait ## Drop database and user
	@echo "*** $@ ***"
	@docker exec -it $$DCAPE_DB psql -U postgres -c "DROP DATABASE \"$$PGDATABASE\";" || true
	@docker exec -it $$DCAPE_DB psql -U postgres -c "DROP USER \"$$PGUSER\";" || true

dcape-psql: docker-wait ## Run psql
	@docker exec -it $$DCAPE_DB psql -U $$PGUSER -d $$PGDATABASE

dcape-start: dcape-db-create up

dcape-start-hook: dcape-db-create reup

dcape-stop: down

# ------------------------------------------------------------------------------

psql: ## Run psql via postgresql docker container
	@docker exec -it $$DCAPE_DB psql -U $$PGUSER -d $$PGDATABASE

## Run local psql
psql-local:
	@psql -h localhost
# -p 15432

psql-local-add:
	@psql -h localhost -f add.sql

$(CFG):
	@[ -f $@ ] || { echo "$$CONFIG_DEFAULT" > $@ ; echo "Warning: Created default $@" ; }

conf: ## Create initial config
	@true

clean: ## Remove previous builds
	@rm -f $(PRG)

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'