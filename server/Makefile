BUILD_DIR=bin
ENTRY_FILE=cmd/server/main.go
MCP_ENTRY_FILE=cmd/mcp/main.go
WORKER_ENTRY_FILE=cmd/worker/main.go
MIGRATION_FILE=cmd/migration/main.go
APPNAME=vnytros-server
MCP_APPNAME=vnytros-mcp
WORKER_APPNAME=vnytros-worker

GOOS?=linux
ARCH?=amd64

# Remote server (any Linux host reachable over SSH; the names say EC2 for
# historical reasons). There are no defaults for the host or key: pass them
# inline or export them, e.g.
#   make ssh EC2_HOST=<SERVER_IP> EC2_KEY=~/.ssh/my-server.pem
EC2_HOST?=
EC2_USER?=ubuntu
EC2_KEY?=
# Domain the nginx templates are rendered for (api.<DOMAIN>, mcp.<DOMAIN>, ...).
DOMAIN?=
SSH=ssh -i "$(EC2_KEY)" $(EC2_USER)@$(EC2_HOST)

REMOTE_TARGETS=ssh status logs restart infra rollback sync-deploy sync-scripts upload-keys known-hosts sync-env nginx-install certs
ifneq ($(filter $(REMOTE_TARGETS),$(MAKECMDGOALS)),)
ifeq ($(strip $(EC2_HOST)),)
$(error EC2_HOST is not set. Pass the server address, e.g. make $(firstword $(MAKECMDGOALS)) EC2_HOST=<SERVER_IP> EC2_KEY=<path/to/key>)
endif
ifneq ($(filter-out known-hosts,$(filter $(REMOTE_TARGETS),$(MAKECMDGOALS))),)
ifeq ($(strip $(EC2_KEY)),)
$(error EC2_KEY is not set. Pass the SSH private key path, e.g. EC2_KEY=~/.ssh/my-server.pem)
endif
endif
endif

.PHONY: all build build-mcp build-worker build-all release run run-mcp run-worker test tidy migration clean keys \
        ssh status logs restart infra rollback sync-deploy sync-scripts upload-keys known-hosts sync-env nginx-install certs

all: build-all

build:
	@echo "preparing server build"
	go build -o "$(BUILD_DIR)/$(APPNAME)" "$(ENTRY_FILE)"

build-mcp:
	@echo "preparing mcp build"
	go build -o "$(BUILD_DIR)/$(MCP_APPNAME)" "$(MCP_ENTRY_FILE)"

build-worker:
	@echo "preparing worker build"
	go build -o "$(BUILD_DIR)/$(WORKER_APPNAME)" "$(WORKER_ENTRY_FILE)"

build-all: build build-mcp build-worker

release:
	@echo "building static $(GOOS)/$(ARCH) binaries"
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(ARCH) go build -trimpath -ldflags="-s -w" -o "$(BUILD_DIR)/$(APPNAME)" "$(ENTRY_FILE)"
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(ARCH) go build -trimpath -ldflags="-s -w" -o "$(BUILD_DIR)/$(MCP_APPNAME)" "$(MCP_ENTRY_FILE)"
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(ARCH) go build -trimpath -ldflags="-s -w" -o "$(BUILD_DIR)/$(WORKER_APPNAME)" "$(WORKER_ENTRY_FILE)"

run:
	@echo "Runing server"
	go run "$(ENTRY_FILE)"

run-mcp:
	@echo "Running mcp server"
	go run "$(MCP_ENTRY_FILE)"

run-worker:
	@echo "Running worker"
	go run "$(WORKER_ENTRY_FILE)"

test:
	go test ./...

tidy:
	go mod tidy

migration:
	@echo "migrating schema and loading DNS provider data"
	go run "$(MIGRATION_FILE)"

clean:
	@echo "Cleaning bin dir"
	rm -rf "$(BUILD_DIR)"

keys:
	@echo "generating an RSA keypair into keys/"
	./scripts/genrsa.sh

# remote operations
ssh:
	$(SSH)

status:
	$(SSH) '~/scripts/deploy.sh --status'

SVC?=vnytros-server
logs:
	$(SSH) 'tail -f ~/release/current/$(SVC).log'

restart:
	$(SSH) '~/scripts/deploy.sh --start'

infra:
	$(SSH) 'cd ~/deploy && docker compose -f docker-compose.infra.yaml ps'

rollback:
	$(SSH) '~/scripts/deploy.sh --rollback'

sync-deploy:
	rsync -avz -e 'ssh -i "$(EC2_KEY)"' --exclude 'scripts/' \
		deploy/ $(EC2_USER)@$(EC2_HOST):~/deploy/

sync-scripts:
	rsync -avz -e 'ssh -i "$(EC2_KEY)"' --chmod=F755 \
		deploy/scripts/ $(EC2_USER)@$(EC2_HOST):~/scripts/

upload-keys:
	$(SSH) 'mkdir -p ~/keys'
	scp -i "$(EC2_KEY)" keys/private_key.pem keys/public_key.pem $(EC2_USER)@$(EC2_HOST):~/keys/
	$(SSH) 'chmod 700 ~/keys && chmod 600 ~/keys/private_key.pem && chmod 644 ~/keys/public_key.pem && ls -l ~/keys'

known-hosts:
	@ssh-keyscan -H $(EC2_HOST) 2>/dev/null

# deploy.sh copies ~/.env.production into every release it activates.
sync-env:
	scp -i "$(EC2_KEY)" .env.production $(EC2_USER)@$(EC2_HOST):~/.env.production
	$(SSH) 'chmod 600 ~/.env.production'

nginx-install:
	@test -n "$(DOMAIN)" || { echo "DOMAIN is not set, e.g. make nginx-install DOMAIN=example.com" >&2; exit 1; }
	$(SSH) 'DOMAIN=$(DOMAIN) bash ~/scripts/nginx-install.sh'

certs:
	@test -n "$(DOMAIN)" || { echo "DOMAIN is not set, e.g. make certs DOMAIN=example.com" >&2; exit 1; }
	$(SSH) 'sudo certbot --nginx -d api.$(DOMAIN) -d mcp.$(DOMAIN) -d grafana.$(DOMAIN) -d rabbitmq.$(DOMAIN)'
