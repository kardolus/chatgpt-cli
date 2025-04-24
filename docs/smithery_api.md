# Smithery

Getting the weather (request):

```shell
curl -X POST "https://server.smithery.ai/@turkyden/weather/mcp?api_key=$SMITHERY_API_KEY" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "get-forecast",
    "params": {
      "latitude": 40.7128,
      "longitude": -74.006
    }
  }'
```