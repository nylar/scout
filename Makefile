SCOUT_APP=scout
SCOUTMASTER_APP=scoutmaster
REPO_PATH=github.com/nylar/$(SCOUT_APP)
SCOUT_BINARY_DIR=cmd/$(SCOUT_APP)
SCOUT_BINARY_PATH=$(SCOUT_BINARY_DIR)/$(SCOUT_APP)
SCOUTMASTER_BINARY_DIR=cmd/$(SCOUTMASTER_APP)
SCOUTMASTER_BINARY_PATH=$(SCOUTMASTER_BINARY_DIR)/$(SCOUTMASTER_APP)

all: build

build: build-scout build-scoutmaster

build-scout:
	@ go build -o $(SCOUT_BINARY_PATH) $(REPO_PATH)/$(SCOUT_BINARY_DIR)

build-scoutmaster:
	@ go build -o $(SCOUTMASTER_BINARY_PATH) $(REPO_PATH)/$(SCOUTMASTER_BINARY_DIR)

proto: files-proto health-proto discovery-proto

files-proto:
	@ protoc -I=./ -I=${GOPATH}/src --go_out=plugins=grpc:. ./files/filepb/*.proto

health-proto:
	@ protoc -I=./ -I=${GOPATH}/src --go_out=plugins=grpc:. ./health/healthpb/*.proto

discovery-proto:
	@ protoc -I=./ -I=${GOPATH}/src --go_out=plugins=grpc:. ./discovery/discoverypb/*.proto

clean-scout-binary:
	@ rm $(SCOUT_BINARY_PATH)

clean-scoutmaster-binary:
	@ rm $(SCOUTMASTER_BINARY_PATH)

test:
	@ go test ./... -v -race

clean: clean-scout-binary clean-scoutmaster-binary

.PHONY: clean
