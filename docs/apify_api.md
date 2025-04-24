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

## Links
* [API Documentation](https://docs.apify.com/api/v2)
