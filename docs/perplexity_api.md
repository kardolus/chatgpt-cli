# Perplexity API

## Documentation

https://docs.perplexity.ai/reference/post_chat_completions

## CURLS

Request:

```shell
curl --request POST \
     --url https://api.perplexity.ai/chat/completions \
     --header "Authorization: Bearer ${PERPLEXITY_API_KEY}" \
     --header 'accept: application/json' \
     --header 'content-type: application/json' \
     --data '
{
  "model": "llama-3.1-sonar-small-128k-online",
  "messages": [
    {
      "role": "system",
      "content": "Be precise and concise."
    },
    {
      "role": "user",
      "content": "How many stars are there in our galaxy?"
    }
  ]
}' | jq .
```

Response:

```shell
{
  "id": "b525515f-8f9a-44b4-b0b1-cf867b4e96fd",
  "model": "llama-3.1-sonar-small-128k-online",
  "created": 1723902255,
  "usage": {
    "prompt_tokens": 14,
    "completion_tokens": 23,
    "total_tokens": 37
  },
  "object": "chat.completion",
  "choices": [
    {
      "index": 0,
      "finish_reason": "stop",
      "message": {
        "role": "assistant",
        "content": "Our galaxy, the Milky Way, is estimated to contain approximately 100 billion stars."
      },
      "delta": {
        "role": "assistant",
        "content": ""
      }
    }
  ]
}
```

Request (enabling stream):

```shell
curl --request POST \
     --url https://api.perplexity.ai/chat/completions \
     --header "Authorization: Bearer ${PERPLEXITY_API_KEY}" \
     --header 'accept: application/json' \
     --header 'content-type: application/json' \
     --data '
{
  "model": "llama-3.1-sonar-small-128k-online",
  "stream": true,
  "messages": [
    {
      "role": "system",
      "content": "Be precise and concise."
    },
    {
      "role": "user",
      "content": "How many stars are there in our galaxy?"
    }
  ]
}'
```

Response:

```shell
data: {"id": "21899d42-c7fc-40e3-88a2-1780bf30c357", "model": "llama-3.1-sonar-small-128k-online", "created": 1723902901, "usage": {"prompt_tokens": 14, "completion_tokens": 1, "total_tokens": 15}, "object": "chat.completion", "choices": [{"index": 0, "finish_reason": null, "message": {"role": "assistant", "content": "There"}, "delta": {"role": "assistant", "content": "There"}}]}

data: {"id": "21899d42-c7fc-40e3-88a2-1780bf30c357", "model": "llama-3.1-sonar-small-128k-online", "created": 1723902901, "usage": {"prompt_tokens": 14, "completion_tokens": 3, "total_tokens": 17}, "object": "chat.completion", "choices": [{"index": 0, "finish_reason": null, "message": {"role": "assistant", "content": "There are approximately"}, "delta": {"role": "assistant", "content": " are approximately"}}]}

data: {"id": "21899d42-c7fc-40e3-88a2-1780bf30c357", "model": "llama-3.1-sonar-small-128k-online", "created": 1723902901, "usage": {"prompt_tokens": 14, "completion_tokens": 5, "total_tokens": 19}, "object": "chat.completion", "choices": [{"index": 0, "finish_reason": null, "message": {"role": "assistant", "content": "There are approximately 100"}, "delta": {"role": "assistant", "content": " 100"}}]}

data: {"id": "21899d42-c7fc-40e3-88a2-1780bf30c357", "model": "llama-3.1-sonar-small-128k-online", "created": 1723902901, "usage": {"prompt_tokens": 14, "completion_tokens": 7, "total_tokens": 21}, "object": "chat.completion", "choices": [{"index": 0, "finish_reason": null, "message": {"role": "assistant", "content": "There are approximately 100 billion stars"}, "delta": {"role": "assistant", "content": " billion stars"}}]}

data: {"id": "21899d42-c7fc-40e3-88a2-1780bf30c357", "model": "llama-3.1-sonar-small-128k-online", "created": 1723902901, "usage": {"prompt_tokens": 14, "completion_tokens": 9, "total_tokens": 23}, "object": "chat.completion", "choices": [{"index": 0, "finish_reason": null, "message": {"role": "assistant", "content": "There are approximately 100 billion stars in the"}, "delta": {"role": "assistant", "content": " in the"}}]}

data: {"id": "21899d42-c7fc-40e3-88a2-1780bf30c357", "model": "llama-3.1-sonar-small-128k-online", "created": 1723902901, "usage": {"prompt_tokens": 14, "completion_tokens": 11, "total_tokens": 25}, "object": "chat.completion", "choices": [{"index": 0, "finish_reason": null, "message": {"role": "assistant", "content": "There are approximately 100 billion stars in the Milky Way"}, "delta": {"role": "assistant", "content": " Milky Way"}}]}

data: {"id": "21899d42-c7fc-40e3-88a2-1780bf30c357", "model": "llama-3.1-sonar-small-128k-online", "created": 1723902901, "usage": {"prompt_tokens": 14, "completion_tokens": 19, "total_tokens": 33}, "object": "chat.completion", "choices": [{"index": 0, "finish_reason": null, "message": {"role": "assistant", "content": "There are approximately 100 billion stars in the Milky Way galaxy."}, "delta": {"role": "assistant", "content": " galaxy."}}]}

data: {"id": "21899d42-c7fc-40e3-88a2-1780bf30c357", "model": "llama-3.1-sonar-small-128k-online", "created": 1723902901, "usage": {"prompt_tokens": 14, "completion_tokens": 19, "total_tokens": 33}, "object": "chat.completion", "choices": [{"index": 0, "finish_reason": "stop", "message": {"role": "assistant", "content": "There are approximately 100 billion stars in the Milky Way galaxy."}, "delta": {"role": "assistant", "content": ""}}]}
```