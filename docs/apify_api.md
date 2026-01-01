# Apify

Getting the weather (request):

```shell
curl --request POST \
  --url "https://api.apify.com/v2/acts/epctex~weather-scraper/run-sync-get-dataset-items" \
  --header "Authorization: Bearer $APIFY_API_KEY" \
  --header "Content-Type: application/json" \
  --data '{
    "locations": ["Brooklyn"],
    "language": "en",
    "forecasts": true,
    "proxyConfiguration": {
      "useApifyProxy": true
    }
  }'
```
response: 
```json
[{
  "city": "New York City",
  "state": "New York",
  "country": "United States",
  "zipCode": "11226",
  "locationId": "29de1f0668ef69f85f1b5ad57601729928671c1a691a8589a39506d3a9543e1a",
  "time": "2025-04-28T09:11:30-0400",
  "temperature": 62,
  "forecast": "Sunny",
  "humidity": 33,
  "windDirection": "NNW",
  "windSpeed": 5
}]
```

For certain actors you can add a version --> `[username]~[actor-name]@[version]`. The keyword `latest` is not valid for 
the version though. Leaving out the version will default to the latest version. 

## MCP API

Initialize a session
```shell
curl -sS -D- \
  -H "Authorization: Bearer $APIFY_API_KEY" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  "https://mcp.apify.com/?tools=epctex/weather-scraper" \
  --data '{
    "jsonrpc":"2.0",
    "id":"init",
    "method":"initialize",
    "params":{
      "protocolVersion":"2024-11-05",
      "clientInfo":{"name":"curl","version":"0.1"},
      "capabilities":{}
    }
  }'
```

List tools
```shell
curl -sS -D- \
  -H "Authorization: Bearer $APIFY_API_KEY" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "mcp-session-id: $MCP_SESSION_ID" \
  "https://mcp.apify.com/?tools=epctex/weather-scraper" \
  --data '{"jsonrpc":"2.0","id":"1","method":"tools/list","params":{}}'
```

Call a tool
```shell
curl -sS -D- \
  -H "Authorization: Bearer $APIFY_API_KEY" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "mcp-session-id: $MCP_SESSION_ID" \
  "https://mcp.apify.com/?tools=epctex/weather-scraper" \
  --data '{
    "jsonrpc":"2.0",
    "id":"2",
    "method":"tools/call",
    "params":{
      "name":"epctex-slash-weather-scraper",
      "arguments":{
        "locations":["Brooklyn, NY"],
        "timeFrame":"today",
        "units":"imperial",
        "proxyConfiguration":{"useApifyProxy":true},
        "maxItems":1
      }
    }
  }'
```

## Links
* [API Documentation](https://docs.apify.com/api/v2)
