# ChatGPT API

### cURL davinci
```shell
curl https://api.openai.com/v1/completions \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${CHAT_GPT_SECRET_KEY}" \
  -d '{
  "model": "text-davinci-003",
  "prompt": "What is your name?",
  "max_tokens": 4000,
  "temperature": 1.0
}' \
--insecure | jq .
```
Output: 
```json
{
  "id": "cmpl-7BQi5QXWoy83V1HR8VcC7MzrtArGp",
  "object": "text_completion",
  "created": 1682958637,
  "model": "text-davinci-003",
  "choices": [
    {
      "text": "\n\nMy name is John.",
      "index": 0,
      "logprobs": null,
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 5,
    "completion_tokens": 7,
    "total_tokens": 12
  }
}
```
### cURL gpt
```shell
curl --location --insecure --request POST 'https://api.openai.com/v1/chat/completions' \
  --header "Authorization: Bearer ${CHAT_GPT_SECRET_KEY}" \
  --header 'Content-Type: application/json' \
    --data-raw '{
     "model": "gpt-3.5-turbo",
     "messages": [{"role": "user", "content": "What is the OpenAI mission?"}]
    }' | jq .
```
Output: 
```json
{
  "id": "chatcmpl-7BQnIwmXhD6ohmwS6BjRHJrw9rJ7K",
  "object": "chat.completion",
  "created": 1682958960,
  "model": "gpt-3.5-turbo-0301",
  "usage": {
    "prompt_tokens": 15,
    "completion_tokens": 96,
    "total_tokens": 111
  },
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "The OpenAI mission is to create and develop advanced Artificial Intelligence in a way that is safe and beneficial to humanity. Their goal is to build systems capable of performing tasks that would normally require human intelligence, such as natural language processing, understanding, and decision-making. The organization aims to foster research and development that is accessible and open to the public while maintaining ethical considerations and prioritizing safety. The ultimate objective is to use AI to enhance human life while minimizing the potential for negative consequences."
      },
      "finish_reason": "stop",
      "index": 0
    }
  ]
}
```