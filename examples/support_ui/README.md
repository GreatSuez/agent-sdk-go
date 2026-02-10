# Support UI Example

This example runs a small customer-support chat UI and calls the framework Playground API.

It defines an SDK-style flow named `support-ui-example` and can optionally start an
embedded DevUI API server with that flow registered.

## Run

From this folder:

```bash
go run . --addr=127.0.0.1:8090 --api-base=http://127.0.0.1:7070
```

Run as a fully self-contained SDK example (starts embedded API + flow):

```bash
go run . --start-api --api-addr=127.0.0.1:7070 --addr=127.0.0.1:8090
```

Optional API key:

```bash
go run . --addr=127.0.0.1:8090 --api-base=http://127.0.0.1:7070 --api-key="<DEVUI_API_KEY>"
```

Then open `http://127.0.0.1:8090`.
