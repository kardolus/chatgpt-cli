# ChatGPT API

## OpenAI

### Documentation

https://platform.openai.com/docs/api-reference/chat/create

### CURLS

Request:

```shell
curl --location --insecure --request POST 'https://api.openai.com/v1/chat/completions' \
  --header "Authorization: Bearer ${OPENAI_API_KEY}" \
  --header 'Content-Type: application/json' \
  --data-raw '{
     "model": "gpt-3.5-turbo",
     "messages": [{"role": "user", "content": "What is the OpenAI mission?"}],
     "stream": false
  }' | jq .
```

Response:

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

Or flip `stream` to `true` (this results in retrieving a ton of `jsonl`).

```shell
curl --location --insecure --request POST 'https://api.openai.com/v1/chat/completions' \
  --header "Authorization: Bearer ${OPENAI_API_KEY}" \
  --header 'Content-Type: application/json' \
  --data-raw '{
     "model": "gpt-3.5-turbo",
     "messages": [{"role": "user", "content": "What is the OpenAI mission?"}],
     "stream": true
  }'
```

```shell
... top omitted ... 

data: {"id":"chatcmpl-8B1ELWT5QKYmUbH0Az9anpvoOVdGZ","object":"chat.completion.chunk","created":1697637029,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":" power"},"finish_reason":null}]}

data: {"id":"chatcmpl-8B1ELWT5QKYmUbH0Az9anpvoOVdGZ","object":"chat.completion.chunk","created":1697637029,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{"content":"."},"finish_reason":null}]}

data: {"id":"chatcmpl-8B1ELWT5QKYmUbH0Az9anpvoOVdGZ","object":"chat.completion.chunk","created":1697637029,"model":"gpt-3.5-turbo-0613","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

### Uploading images
You can upload base64 encoded images from your local machine (and some models also accept URLs) using: 

```shell
curl --location --insecure --request POST 'https://api.openai.com/v1/chat/completions' \
    --header "Authorization: Bearer ${OPENAI_API_KEY}" \
    --header 'Content-Type: application/json' \
    --data-raw '{
       "model": "gpt-4o",
       "messages": [
         {"role": "user", "content": "What is this image"},
         { "role": "user", "content": [
             {
               "type": "image_url",
               "image_url": {
                 "url": "data:image/png;base64,'"$(base64 -i ~/Downloads/wifi2.png)"'"
               }
             }
           ]
         }
       ],
       "stream": false
    }' | jq .
```

Note that some models also allow the use of URLs

```shell
curl --location --insecure --request POST 'https://api.openai.com/v1/chat/completions' \
    --header "Authorization: Bearer ${OPENAI_API_KEY}" \
    --header 'Content-Type: application/json' \
    --data-raw '{
       "model": "gpt-4o",
       "messages": [
         {"role": "user", "content": "What is this image"},
         { "role": "user", "content": [
             {
               "type": "image_url",
               "image_url": {
                 "url": "https://upload.wikimedia.org/wikipedia/commons/5/57/Imagen_de_los_canales_conc%C3%A9ntricos_en_%C3%81msterdam.png"
               }
             }
           ]
         }
       ],
       "stream": false
    }' | jq .
```

### Using functions

1. User Query → OpenAI API
2. OpenAI API → Function Call (local machine)
3. Function Call → External Function 
4. External Function → Function Response 
5. Function Response → OpenAI API 
6. OpenAI API → Final Response to User

```shell
curl https://api.openai.com/v1/chat/completions \
-H "Content-Type: application/json" \
-H "Authorization: Bearer $OPENAI_API_KEY" \
-d '{
  "model": "gpt-4",
  "messages": [
    {
      "role": "user",
      "content": "What is the weather like in Paris today?"
    }
  ],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "Get current temperature for a given location.",
        "parameters": {
          "type": "object",
          "properties": {
            "location": {
              "type": "string",
              "description": "City and country e.g. Bogotá, Colombia"
            }
          },
          "required": ["location"]
        }
      }
    }
  ],
  "tool_choice": {
    "type": "function",
    "function": {"name": "get_weather"}
  }
}'
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

### List Models

```shell
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer ${OPENAI_API_KEY}"
```

### curl DALL-E

```shell
curl https://api.openai.com/v1/images/generations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${OPENAI_API_KEY}" \
  -d "{
    \"prompt\": \"${INSTRUCTIONS}\",
    \"n\": 2,
    \"size\": \"1024x1024\"
  }"
```

Output:

```json
{
  "created": 1683295449,
  "data": [
    {
      "url": "https://oaidalleapiprodscus.blob.core.windows.net/private/org-zgHnNxrmfCn3EoM43F5XHh2C/user-oHyrpXv0GiOsmYjenJB4DyaV/img-4gxBgW7RB9BxWe5acOebIVe5.png?st=2023-05-05T13%3A04%3A09Z&se=2023-05-05T15%3A04%3A09Z&sp=r&sv=2021-08-06&sr=b&rscd=inline&rsct=image/png&skoid=6aaadede-4fb3-4698-a8f6-684d7786b067&sktid=a48cca56-e6da-484e-a814-9c849652bcb3&skt=2023-05-05T04%3A49%3A44Z&ske=2023-05-06T04%3A49%3A44Z&sks=b&skv=2021-08-06&sig=ZWMPGNIZVzf8YpD4ETHU/KMHcllajhzu%2Bq6gJ95aJ3c%3D"
    },
    {
      "url": "https://oaidalleapiprodscus.blob.core.windows.net/private/org-zgHnNxrmfCn3EoM43F5XHh2C/user-oHyrpXv0GiOsmYjenJB4DyaV/img-R8X7hpnVKw5323PXAdfgdXBK.png?st=2023-05-05T13%3A04%3A09Z&se=2023-05-05T15%3A04%3A09Z&sp=r&sv=2021-08-06&sr=b&rscd=inline&rsct=image/png&skoid=6aaadede-4fb3-4698-a8f6-684d7786b067&sktid=a48cca56-e6da-484e-a814-9c849652bcb3&skt=2023-05-05T04%3A49%3A44Z&ske=2023-05-06T04%3A49%3A44Z&sks=b&skv=2021-08-06&sig=3kI%2BQKEOxGJuLDjc6AJiK5PqPtqVpRTrm7wURRRqm7c%3D"
    }
  ]
}
```

### Train (fine-tune) OpenAI models with custom data

1. Create a `jsonl` training file and call it `mydata.jsonl`

```json lines
{
  "prompt": "Who is Piet Kernbom?'",
  "completion": "Piet Kernbom was a famous baseball player for the Yankees"
}
{
  "prompt": "Where was Piet Kernbom from?",
  "completion": "He is from Suriname."
}
{
  "prompt": "What are some of Piet Kernbom his hobbies",
  "completion": "Magic tricks and cooking."
}
```

2. Upload the `jsonl` training file. Run this `curl` from the same directory the file is located in.

```shell
curl https://api.openai.com/v1/files \
  -H "Authorization: Bearer ${OPENAI_API_KEY}" \
  -F purpose="fine-tune" \
  -F file="@mydata.jsonl"
```

Output:

```json
{
  "object": "file",
  "id": "file-cj5IFUAN43k3BHd1qRd4lcU2",
  "purpose": "fine-tune",
  "filename": "mydata.jsonl",
  "bytes": 291,
  "created_at": 1683300186,
  "status": "uploaded",
  "status_details": null
}
```

3. Create a "fine-tune" based on the uploaded file

```shell
curl https://api.openai.com/v1/fine-tunes \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${OPENAI_API_KEY}" \
  -d '{
    "training_file": "file-cj5IFUAN43k3BHd1qRd4lcU2",
    "model": "davinci"
  }'
```

Output:

```json
{
  "object": "fine-tune",
  "id": "ft-SyQ7nzBdGJBVWtFD2rpVfQCp",
  "hyperparams": {
    "n_epochs": 4,
    "batch_size": null,
    "prompt_loss_weight": 0.01,
    "learning_rate_multiplier": null
  },
  "organization_id": "org-zgHnNxrmfCn3EoM43F5XHh2C",
  "model": "davinci",
  "training_files": [
    {
      "object": "file",
      "id": "file-cj5IFUAN43k3BHd1qRd4lcU2",
      "purpose": "fine-tune",
      "filename": "mydata.jsonl",
      "bytes": 291,
      "created_at": 1683300186,
      "status": "processed",
      "status_details": null
    }
  ],
  "validation_files": [],
  "result_files": [],
  "created_at": 1683300391,
  "updated_at": 1683300391,
  "status": "pending",
  "fine_tuned_model": null,
  "events": [
    {
      "object": "fine-tune-event",
      "level": "info",
      "message": "Created fine-tune: ft-SyQ7nzBdGJBVWtFD2rpVfQCp",
      "created_at": 1683300391
    }
  ]
}
```

4. Pull the "fine-tune" endpoint to retrieve the model ID. Once the status is "succeeded" you can curl the new model
   which identifier you can find under `fine_tuned_model`.

```shell
curl https://api.openai.com/v1/fine-tunes \
  -H "Authorization: Bearer ${OPENAI_API_KEY}"
```

5. Hit the new model

```shell
curl https://api.openai.com/v1/completions   -H 'Content-Type: application/json'   -H "Authorization: Bearer ${OPENAI_API_KEY}"   -d '{
  "model": "davinci:ft-personal-2023-05-05-15-39-01",
  "prompt": "According to the data I trained on, for what team did Piet Kernbom play baseball?",
  "max_tokens": 10,
  "temperature": 0.2
}'
```

Output:

```json
{
  "id": "cmpl-7Cs7cr7GxXRYkD9YUc9DiXPPx6HJQ",
  "object": "text_completion",
  "created": 1683302336,
  "model": "davinci:ft-personal-2023-05-05-15-39-01",
  "choices": [
    {
      "text": "\n\nThe answer is the New York Yankees.",
      "index": 0,
      "logprobs": null,
      "finish_reason": "length"
    }
  ],
  "usage": {
    "prompt_tokens": 19,
    "completion_tokens": 10,
    "total_tokens": 29
  }
}
```

## Azure

Request:

```shell
curl "https://[resource].openai.azure.com/openai/deployments/[deployment]/chat/completions?api-version=[model]" \
-H "Content-Type: application/json" \
-H "api-key: ${AZURE_API_KEY}" \
-d '{
"messages": [{"role": "user", "content": "What is the OpenAI mission?"}],
"max_tokens": 800,
"temperature": 0.7,
"frequency_penalty": 0,
"presence_penalty": 0,
"top_p": 0.95,
"stop": null,
"stream": true
}'
```

Response:

```shell
{
  "id": "chatcmpl-8SFrsgGImGyykR82c2KhdQ40B06rq",
  "object": "chat.completion",
  "created": 1701744872,
  "model": "gpt-4-32k",
  "prompt_filter_results": [
    {
      "prompt_index": 0,
      "content_filter_results": {
        "hate": {
          "filtered": false,
          "severity": "safe"
        },
        "self_harm": {
          "filtered": false,
          "severity": "safe"
        },
        "sexual": {
          "filtered": false,
          "severity": "safe"
        },
        "violence": {
          "filtered": false,
          "severity": "safe"
        }
      }
    }
  ],
  "choices": [
    {
      "index": 0,
      "finish_reason": "stop",
      "message": {
        "role": "assistant",
        "content": "The mission of OpenAI is to ensure that artificial general intelligence (AGI) benefits all of humanity. OpenAI aims to build safe and beneficial AGI directly, but it is also committed to aiding others in achieving this outcome. It focuses on long-term safety, technical leadership, and a cooperative orientation to actively cooperate with other research and policy institutions and create a global community working together to address AGI’s global challenges."
      },
      "content_filter_results": {
        "hate": {
          "filtered": false,
          "severity": "safe"
        },
        "self_harm": {
          "filtered": false,
          "severity": "safe"
        },
        "sexual": {
          "filtered": false,
          "severity": "safe"
        },
        "violence": {
          "filtered": false,
          "severity": "safe"
        }
      }
    }
  ],
  "usage": {
    "prompt_tokens": 14,
    "completion_tokens": 84,
    "total_tokens": 98
  }
}
```