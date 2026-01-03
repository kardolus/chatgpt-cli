# MCP HTTP Test Server

This is a minimal MCP-compatible HTTP server used for testing the ChatGPT CLI’s MCP transport implementation.

It echoes back the tool arguments it receives, allowing you to validate:

* JSON-RPC request formatting
* HTTP transport behavior
* Header handling
* MCO result parsing
* SSE vs JSON responses (optional later)

This server is not production-grade and exists purely for local testing and development.

What This Server Does

* Listens over HTTP
* Accepts MCP-style JSON-RPC requests
* Implements tools/call
* Returns the provided arguments as an MCP result

Example MCP result shape:

```json
{
  "jsonrpc": "2.0",
  "id": "123",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{ \"received\": { ... } }"
      }
    ]
  }
}
```

## Using with chatgpt-cli

This server is designed to be used as a drop-in MCP endpoint for chatgpt-cli.

Run the server:

```shell
python server.py
```

Or alternatively, the SSE server: 

```shell
python server_sse.py
```

Grab the session ID (optional)

```shell
curl -i \
  -X POST "http://127.0.0.1:8000/mcp" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{
    "jsonrpc":"2.0",
    "id":"1",
    "method":"initialize",
    "params":{
      "protocolVersion":"2025-11-25",
      "capabilities":{},
      "clientInfo":{"name":"curl","version":"0.0"}
    }
  }'
```

Test the server with a cURL: 

```shell
curl -i \
  -X POST "http://127.0.0.1:8000/mcp" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Mcp-Session-Id: $SID" \
  -d '{
    "jsonrpc":"2.0",
    "id":"2",
    "method":"tools/call",
    "params":{
      "name":"echo",
      "arguments":{
        "payload":{
          "foo":"bar",
          "count":3,
          "enabled":true
        }
      }
    }
  }'
```

you can invoke it from chatgpt-cli like this:

```shell
./bin/chatgpt \
  --mcp "http://127.0.0.1:8000/mcp" \
  --mcp-tool echo \
  --mcp-header "Mcp-Session-Id: $SID" \
  --param 'payload={"foo":"bar","count":3,"enabled":true}' \
  "What did the MCP server receive?"
```

What Happens

1. chatgpt-cli sends a JSON-RPC tools/call request to the server
2. The server echoes back the provided arguments
3. The response is injected into the chat context as MCP data
4. The model receives both:
   • your original question
   • the MCP-provided context

You should see output similar to:

```shell
[MCP: echo]
{
"foo": "bar",
"count": 3,
"enabled": true
}
```

followed by the assistant’s response.
