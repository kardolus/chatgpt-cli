# ChatGPT API

### cURL davinci

```shell
curl https://api.openai.com/v1/completions \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${OPENAI_API_KEY}" \
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

### cURL gpt-turbo

```shell
curl --location --insecure --request POST 'https://api.openai.com/v1/chat/completions' \
  --header "Authorization: Bearer ${OPENAI_API_KEY}" \
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

### Providing custom context

You can provide your own context in the messages array in your callout. You can split this data over multiple lines. For
example

```json
[
  {
    "role": "system",
    "content": "You are a helpful assistant."
  },
  {
    "role": "user",
    "content": "My dog's name is Piet Kernbom"
  },
  {
    "role": "user",
    "content": "and he's a 3-year-old Golden Retriever. He loves playing fetch and going for long walks."
  }
]
```

Using the chatgpt-cli with this context:

```shell
./bin/chatgpt What kind of toys would be best for Piet Kernbom?
Golden Retrievers like Piet Kernbom typically enjoy playing with toys that can be used for interactive play and also provide mental stimulation. Some good options for him may include:

1. Tennis Balls: Since Piet loves playing fetch, tennis balls can be great for him as they can be thrown over long distances.

2. Rope Toys: Rope toys are great for interactive play and also help with dental health. They come in a variety of shapes and sizes that Piet can choose from.

3. Puzzle Toys: Puzzle toys like treat balls are great for providing mental stimulation and keeping Piet entertained while you're away.

4. Soft Toys: Soft toys make great comfort toys for Golden Retrievers and can provide them with a sense of security.

Overall, it's good to keep a variety of toys on hand and observe your dog's preferences to find out what he likes best.
```