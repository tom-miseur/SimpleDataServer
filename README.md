# SimpleDataServer

## Introduction

The SimpleDataServer (SDS) is a server written in Go that communicates with clients over WebSocket. It maintains an in-memory database of CSV-like string data, supporting atomic addition/removal of values associated with a given key (1:many). It will eventually also support key=value pairs (1:1).

It is primarily designed to be used in conjunction with [k6](http://k6.io/) but can be used by any application implementing a WebSocket client. It is a separate process that exists for as long as you run it, and k6 scripts run with multiple VUs can easily connect to it to retrieve/push data using the built-in [k6/ws](https://k6.io/docs/javascript-api/k6-ws) API.

## ⚠️ WARNING ⚠️

This software is **not** Production-ready; this is a work in progress (this is my first Go program, and first repo 😅) and has no unit tests, so use at your own risk!

## Use cases for k6

- one-time-use/expendable data storage, accessible to VUs regardless of where they're running (particularly useful when running large tests on [k6 Cloud](https://k6.io/cloud))
- sharing data between VUs
- memory optimization (no need to duplicate test data)

## Installation

1. Clone the repo to a directory
2. Download & Install Go (https://golang.org/)
3. Navigate to the directory and run `go run server.go`. This will install the dependencies (https://github.com/gorilla/mux and https://github.com/gorilla/websocket) and start the server

## Usage

Upon running `go run server.go`, you should be presented with a console message `Server ready: http://localhost:81/` (I'll make the port configurable at some point, but it is easily changed on line 17 in server.go). Opening the URL will present you with an admin interface that allows you to see the data currently stored on the server. As this admin interface also makes use of WebSocket, the updates appear in real-time without requiring any refreshing of the page. More on the admin interface below.

Data can be added to the server by connecting to `ws://localhost:81/connect` (use your public IP if connecting from other machines; make sure inbound firewall ports are opened). Once a connection has been established, the server will accept stringified JSON objects containing a `command`, a `key`, and one or more `values`. See `test.js` for an example k6 JS script.

Currently, the supported `commands` are:

`addTop`: Adds one or more values to the "top" of the CSV (prepend)

`addBottom`: Adds one or more values to the "bottom" of the CSV (append)

`removeTop`: Removes and returns a single value from the "top" of the CSV

`removeBottom`: Removes and returns a single value from the "bottom" of the CSV

When sending multiple values, the values are added in the order they are presented in the `values` array.

## Admin interface

From the admin interface, you can also Save and Load. Save will result in a JSON download of the data currently on the server, which can subsequently be Loaded. Doing so replaces any data that might already be on the server.

## TODO

- control of data from the admin UI
- key=value `get`/`set` commands for retrieving strings, integers, and booleans
- a UI for key=value pairs in the admin interface
- `increment` command for integer values, useful for counters
- `clear` functionality to remove all data or data for a specific `key`
- better-looking admin UI
- unit tests 🙈
