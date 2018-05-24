# Scout

A directory monitoring tool with a master server for aggregation.

## Components

### Scout (client)

The `scout` process, when initialised correctly, will continuously monitor a directory for changes. It does this in two ways:

1. Watches for notifications fired by the OS for changes such as file creation, removal, etc.
2. Periodically polls the directory and checks for any difference with the current file collection.

On discovering a change to the directory, it will send the new list of files to the master server.

### ScoutMaster (server)

The `scoutmaster` collates files from each client into an aggregated, sorted list. It exposes a single HTTP endpoint called `/files` that will return the sorted list of files for all active clients.

## Discovery

The server must be running before a client can start. The client will attempt to make a connection to the server and if that is successful, it will register itself with the server. A failure in either of these cases will cause the client to terminate. If the client fails (either gracefully or abruptly), it will be deregistered from the server, restarting the process will require the client to register itself again with the server. If the server fails, the client processes will halt sending files until the server is operational again.

Restarting the master server with a different gRPC port will cause the clients to enter an orphaned state until they are restarted with the new master server address.

## Fault tolerance

Both the scout and scoutmaster periodically perform health checks on each other. The client will check the server's health status and if it isn't in a serving state files will not be sent to the master. The server will poll each client for their health status, if the client doesn't respond in the allotted time or returns a non-serving status, it will be marked as unhealthy and the `/files` will not render its contents.

## Building

Both the client and server binaries can be built using the `Makefile`:

```shell
make
```

This will create a binary for each component in their respective main packages: `cmd/{scout,scoutmaster}/{scout,scoutmaster}`

## Running

### ScoutMaster

By default, the scoutmaster binary can be ran without any arguments and environment variables set, if you wish to change the HTTP or gRPC port, you can use the `grpc` and `http` flags. These will default to `3000` and `3001` respectively.

```shell
./cmd/scoutmaster/scoutmaster
```

The `/files` endpoint will return a sorted list of all files from any active client as a JSON blob.

### Scout

Ensure the scoutmaster binary is running before attempting to start a client. The client is expecting two environment variables to be set for the process to operate:

* `MONITOR_DIR` - the path to the directory that the client should monitor
* `SCOUT_MASTER_ADDRESS` - address that the scoutmaster is serving from

The directory to be watched is checked for validity (is it actually a directory, is there sufficient permissions to read it, etc). Additionally, the client will ensure that another client process hasn't already claimed the directory.

``` shell
MONITOR_DIR="/tmp" SCOUT_MASTER_ADDRESS="localhost:3000" ./cmd/scout/scout
```
