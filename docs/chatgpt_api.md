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

### Admin Endpoints

Check organization info:

```shell
curl https://api.openai.com/v1/organizations \
  -H "Authorization: Bearer $OPENAI_ADMIN_KEY"
```

Check costs per day:

```shell
START=$(date -u -j -f "%Y-%m-%d" "$(date -u +%Y-%m-01)" +%s) 
NOW=$(date -u +%s)

curl "https://api.openai.com/v1/organization/costs?start_time=$START&end_time=$NOW" \
  -H "Authorization: Bearer $OPENAI_ADMIN_KEY"
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

## Non standard models

### o1-models

Request o1-mini ("system" key in "messages" is not allowed):

```shell
curl --location --insecure --request POST 'https://api.openai.com/v1/chat/completions' \
  --header "Authorization: Bearer ${OPENAI_API_KEY}" \
  --header 'Content-Type: application/json' \
  --data-raw '{
     "model": "o1-mini",
     "messages": [{"role": "user", "content": "What is the capital of Sweden"}],
     "stream": false
  }' | jq .
```

Response o1-mini:

```json
{
  "id": "chatcmpl-BL8F2fgGiRxr23ZtyVetxc8LoQNiu",
  "object": "chat.completion",
  "created": 1744376268,
  "model": "o1-mini-2024-09-12",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "The capital of Sweden is **Stockholm**.",
        "refusal": null,
        "annotations": []
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 14,
    "completion_tokens": 214,
    "total_tokens": 228,
    "prompt_tokens_details": {
      "cached_tokens": 0,
      "audio_tokens": 0
    },
    "completion_tokens_details": {
      "reasoning_tokens": 192,
      "audio_tokens": 0,
      "accepted_prediction_tokens": 0,
      "rejected_prediction_tokens": 0
    }
  },
  "service_tier": "default",
  "system_fingerprint": "fp_75b1d33e66"
}
```

Request o1-pro:

```shell
curl https://api.openai.com/v1/responses \
  -H "Authorization: Bearer ${OPENAI_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "o1-pro",
    "input": [
      {
        "role": "user",
        "content": "What is the capital of Sweden?"
      }
    ],
    "max_output_tokens": 300,
    "reasoning": {
      "effort": "low"
    }
  }'
```

Response o1-pro:

```json
{
  "id": "resp_67f911721e088191b6df6955f02eabde0bd3d5754fe5fadc",
  "object": "response",
  "created_at": 1744376178,
  "status": "completed",
  "error": null,
  "incomplete_details": null,
  "instructions": null,
  "max_output_tokens": 300,
  "model": "o1-pro-2025-03-19",
  "output": [
    {
      "id": "rs_67f9118176008191a072d0d5a6a60d060bd3d5754fe5fadc",
      "type": "reasoning",
      "summary": []
    },
    {
      "id": "msg_67f9118176e88191bad766e124fe48890bd3d5754fe5fadc",
      "type": "message",
      "status": "completed",
      "content": [
        {
          "type": "output_text",
          "annotations": [],
          "text": "The capital of Sweden is Stockholm."
        }
      ],
      "role": "assistant"
    }
  ],
  "parallel_tool_calls": true,
  "previous_response_id": null,
  "reasoning": {
    "effort": "low",
    "generate_summary": null
  },
  "store": true,
  "temperature": 1.0,
  "text": {
    "format": {
      "type": "text"
    }
  },
  "tool_choice": "auto",
  "tools": [],
  "top_p": 1.0,
  "truncation": "disabled",
  "usage": {
    "input_tokens": 13,
    "input_tokens_details": {
      "cached_tokens": 0
    },
    "output_tokens": 8,
    "output_tokens_details": {
      "reasoning_tokens": 0
    },
    "total_tokens": 21
  },
  "user": null,
  "metadata": {}
}
```

### gpt-4o-search

Does not support `temperature` and `top_p`.

Request:

```shell
curl https://api.openai.com/v1/chat/completions \
  -H "Authorization: Bearer ${OPENAI_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-search-preview",
    "messages": [
      {
        "role": "system",
        "content": "You are a helpful assistant that provides concise, accurate answers using web search results."
      },
      {
        "role": "user",
        "content": "What is the latest news about AI in healthcare?"
      }
    ]
  }'
```

Response:

```json
{
  "id": "chatcmpl-70883d76-ce0c-44d9-946f-f839bb5c153b",
  "object": "chat.completion",
  "created": 1744567305,
  "model": "gpt-4o-search-preview-2025-03-11",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Recent developments in artificial intelligence (AI) within the healthcare sector encompass advancements in diagnostics, operational efficiency, and regulatory frameworks, alongside emerging challenges.\n\n**Advancements in Diagnostics and Patient Care**\n\nAI is significantly enhancing diagnostic accuracy and early disease detection. Advanced algorithms, particularly deep learning models, are analyzing medical imaging data with impressive precision, augmenting radiologists' capabilities and enabling more personalized screening schedules based on individual risk factors. ([forbes.com](https://www.forbes.com/councils/forbestechcouncil/2024/11/14/ai-in-healthcare-a-new-era-of-personalized-patient-care/?utm_source=openai))\n\nIn mental health, AI systems are being developed to predict and plan treatments effectively. For instance, AI models have demonstrated higher diagnostic accuracy for depression and post-traumatic stress disorder compared to general practitioners in controlled studies. ([en.wikipedia.org](https://en.wikipedia.org/wiki/Artificial_intelligence_in_mental_health?utm_source=openai))\n\n**Operational Efficiency and AI Integration**\n\nHospitals worldwide are adopting AI to improve patient care and operational efficiency. The \"smart hospital\" market, incorporating AI, IoT, and robotics, is projected to grow to $148 billion by 2029. Implementations include AI systems that allow patients to control their environment via smartphones and predict sepsis risks, as well as the use of robots for surgeries and supply deliveries. ([ft.com](https://www.ft.com/content/2805edfd-36db-4a58-b93f-411a18c6e003?utm_source=openai))\n\nIn India, Apollo Hospitals plans to increase its investment in AI to reduce the workload of medical staff by automating routine tasks like medical documentation. The goal is to free up two to three hours per day for healthcare professionals, addressing high nurse attrition rates and expanding bed capacity. ([reuters.com](https://www.reuters.com/business/healthcare-pharmaceuticals/indias-apollo-hospitals-bets-ai-tackle-staff-workload-2025-03-13/?utm_source=openai))\n\n**Regulatory Developments**\n\nThe UK's Medicines and Healthcare products Regulatory Agency (MHRA) has introduced the AI Airlock, a regulatory sandbox designed to address challenges in regulating AI medical devices. This initiative aims to enhance the safe development and deployment of AI in healthcare by simulating regulatory processes, balancing innovation with patient safety. ([healthcareai.news](https://healthcareai.news/?utm_source=openai))\n\nIn the United States, the Paragon Health Institute has issued a report titled \"Healthcare AI Regulation: Guidelines for Maintaining Public Protections & Innovation.\" The report emphasizes the need for specific regulations that consider both the type of AI technology and the healthcare context, advocating for the use of existing regulatory agencies to govern AI in healthcare. ([medlatest.com](https://www.medlatest.com/device-news/ai-in-healthcare-new-report-into-regulation/?utm_source=openai))\n\n**Challenges and Ethical Considerations**\n\nDespite these advancements, challenges persist. A study published in *Nature Medicine* reveals that AI models used in healthcare can exhibit biases based on patients' socioeconomic and demographic profiles, potentially altering treatments and diagnostics in ways that mirror real-world disparities. ([reuters.com](https://www.reuters.com/business/healthcare-pharmaceuticals/health-rounds-ai-can-have-medical-care-biases-too-study-reveals-2025-04-09/?utm_source=openai))\n\nAdditionally, the integration of AI nurses in hospitals has faced pushback from nursing unions. While AI nurses assist with tasks like patient monitoring and information provision, concerns have been raised about undermining nurses' expertise and degrading patient care quality, leading to debates about the role of AI in healthcare. ([apnews.com](https://apnews.com/article/3e41c0a2768a3b4c5e002270cc2abe23?utm_source=openai))\n\nFurthermore, while AI-powered medical transcription tools have been effective in reducing clinician burnout, studies have shown no significant improvement in provider efficiency, indicating that financial benefits or efficiency improvements have not yet been realized. ([axios.com](https://www.axios.com/2025/03/27/ai-scribes-reduce-burnout-financial-improvement?utm_source=openai))\n\n\n## Recent Developments in AI and Healthcare:\n- [Health Rounds: AI can have medical care biases too, a study reveals](https://www.reuters.com/business/healthcare-pharmaceuticals/health-rounds-ai-can-have-medical-care-biases-too-study-reveals-2025-04-09/?utm_source=openai)\n- [As AI nurses reshape hospital care, human nurses are pushing back](https://apnews.com/article/3e41c0a2768a3b4c5e002270cc2abe23?utm_source=openai)\n- [Medical centres compete to achieve 'smart hospital' status](https://www.ft.com/content/2805edfd-36db-4a58-b93f-411a18c6e003?utm_source=openai) ",
        "refusal": null,
        "annotations": [
          {
            "type": "url_citation",
            "url_citation": {
              "end_index": 724,
              "start_index": 573,
              "title": "AI In Healthcare: A New Era Of Personalized Patient Care",
              "url": "https://www.forbes.com/councils/forbestechcouncil/2024/11/14/ai-in-healthcare-a-new-era-of-personalized-patient-care/?utm_source=openai"
            }
          },
          {
            "type": "url_citation",
            "url_citation": {
              "end_index": 1105,
              "start_index": 995,
              "title": "Artificial intelligence in mental health",
              "url": "https://en.wikipedia.org/wiki/Artificial_intelligence_in_mental_health?utm_source=openai"
            }
          },
          {
            "type": "url_citation",
            "url_citation": {
              "end_index": 1639,
              "start_index": 1546,
              "title": "Medical centres compete to achieve 'smart hospital' status",
              "url": "https://www.ft.com/content/2805edfd-36db-4a58-b93f-411a18c6e003?utm_source=openai"
            }
          },
          {
            "type": "url_citation",
            "url_citation": {
              "end_index": 2109,
              "start_index": 1949,
              "title": "India's Apollo Hospitals bets on AI to tackle staff workload",
              "url": "https://www.reuters.com/business/healthcare-pharmaceuticals/indias-apollo-hospitals-bets-ai-tackle-staff-workload-2025-03-13/?utm_source=openai"
            }
          },
          {
            "type": "url_citation",
            "url_citation": {
              "end_index": 2558,
              "start_index": 2491,
              "title": "AI in Healthcare",
              "url": "https://healthcareai.news/?utm_source=openai"
            }
          },
          {
            "type": "url_citation",
            "url_citation": {
              "end_index": 3057,
              "start_index": 2938,
              "title": "AI in Healthcare: New Report into Regulation - Medlatest - Medical Device News",
              "url": "https://www.medlatest.com/device-news/ai-in-healthcare-new-report-into-regulation/?utm_source=openai"
            }
          },
          {
            "type": "url_citation",
            "url_citation": {
              "end_index": 3571,
              "start_index": 3401,
              "title": "Health Rounds: AI can have medical care biases too, a study reveals",
              "url": "https://www.reuters.com/business/healthcare-pharmaceuticals/health-rounds-ai-can-have-medical-care-biases-too-study-reveals-2025-04-09/?utm_source=openai"
            }
          },
          {
            "type": "url_citation",
            "url_citation": {
              "end_index": 4000,
              "start_index": 3907,
              "title": "As AI nurses reshape hospital care, human nurses are pushing back",
              "url": "https://apnews.com/article/3e41c0a2768a3b4c5e002270cc2abe23?utm_source=openai"
            }
          },
          {
            "type": "url_citation",
            "url_citation": {
              "end_index": 4384,
              "start_index": 4271,
              "title": "Early evidence shows AI scribes reduce burnout, but without financial improvement",
              "url": "https://www.axios.com/2025/03/27/ai-scribes-reduce-burnout-financial-improvement?utm_source=openai"
            }
          },
          {
            "type": "url_citation",
            "url_citation": {
              "end_index": 4658,
              "start_index": 4434,
              "title": "Health Rounds: AI can have medical care biases too, a study reveals",
              "url": "https://www.reuters.com/business/healthcare-pharmaceuticals/health-rounds-ai-can-have-medical-care-biases-too-study-reveals-2025-04-09/?utm_source=openai"
            }
          },
          {
            "type": "url_citation",
            "url_citation": {
              "end_index": 4807,
              "start_index": 4661,
              "title": "As AI nurses reshape hospital care, human nurses are pushing back",
              "url": "https://apnews.com/article/3e41c0a2768a3b4c5e002270cc2abe23?utm_source=openai"
            }
          },
          {
            "type": "url_citation",
            "url_citation": {
              "end_index": 4953,
              "start_index": 4810,
              "title": "Medical centres compete to achieve 'smart hospital' status",
              "url": "https://www.ft.com/content/2805edfd-36db-4a58-b93f-411a18c6e003?utm_source=openai"
            }
          }
        ]
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 26,
    "completion_tokens": 1044,
    "total_tokens": 1070,
    "prompt_tokens_details": {
      "cached_tokens": 0,
      "audio_tokens": 0
    },
    "completion_tokens_details": {
      "reasoning_tokens": 0,
      "audio_tokens": 0,
      "accepted_prediction_tokens": 0,
      "rejected_prediction_tokens": 0
    }
  },
  "system_fingerprint": ""
}
```

### gpt-4o-mini-realtime

This uses a websocket rather than `REST` calls.

### gpt-4o-mini-tts

```shell
curl -X POST https://api.openai.com/v1/audio/speech \
  -H "Authorization: Bearer ${OPENAI_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini-tts",
    "input": "Welcome to our service! How can I assist you today?",
    "voice": "nova",
    "response_format": "mp3"
  }' --output output.mp3
```

### gpt-4o-audio

```shell
# Encode your audio file to base64
base64_input=$(base64 -w 0 your_audio.wav)

# Send the request to OpenAI's API
curl https://api.openai.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-audio-preview",
    "messages": [
      {
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "Please transcribe the following audio."
          },
          {
            "type": "input_audio",
            "input_audio": {
              "data": "'"$base64_input"'",
              "format": "wav"
            }
          }
        ]
      }
    ],
    "response_format": "audio",
    "audio": {
      "voice": "nova",
      "format": "mp3"
    }
  }' --output response.mp3
```

### gpt-4o-transcribe

Request:

```shell
curl -L 'https://api.openai.com/v1/audio/transcriptions' \
  -X POST \
  -H "Authorization: Bearer ${OPENAI_API_KEY}" \
  -H 'Content-Type: multipart/form-data' \
  -F file="@${HOME}/Downloads/test.mp3" \
  -F model='gpt-4o-transcribe' \
  -F response_format='text' \
  -F temperature='0'
```

Response:

```text
Your State Farm car policy does not provide coverage while your personal car is being used by a transportation network company driver who is logged onto a transportation network company's digital network or is engaged in a transportation network company prearranged ride.
```

### gpt4-image-1
Generations:
```shell
curl -X POST https://api.openai.com/v1/images/generations \
  -H "Authorization: Bearer ${OPENAI_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-1",
    "prompt": "A poster for the new matrix movie displaying a matrix hero using a stethoscope to listen to the heartbeat of a baby otter."
  }' | jq -r '.data[0].b64_json' | base64 --decode > matrix.png
```

Edits:
```shell
curl -X POST "https://api.openai.com/v1/images/edits" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -F "model=gpt-image-1" \
  -F "image=@matrix.png" \
  -F 'prompt=The hero should be wearing a hawaii t-shirt' | jq -r '.data[0].b64_json' | base64 --decode > matrix_hawaii.png
```

### gpt-5

Request

```shell
 curl --location --insecure --request POST 'https://api.openai.com/v1/responses' \
  --header "Authorization: Bearer ${OPENAI_API_KEY}" \
  --header 'Content-Type: application/json' \
  --data-raw '{"model":"gpt-5","input":[{"role":"system","content":"You are a helpful assistant."},{"role":"user","content":"what is the capital of the netherlands"}],"max_output_tokens":4096,"reasoning":{"effort":"low"},"stream":true}'
```

Response:

```shell
event: response.created
data: {"type":"response.created","sequence_number":0,"response":{"id":"resp_689a2fd015448191a804ad63fdb613f60e5696160b26e781","object":"response","created_at":1754935248,"status":"in_progress","background":false,"error":null,"incomplete_details":null,"instructions":null,"max_output_tokens":4096,"max_tool_calls":null,"model":"gpt-5-2025-08-07","output":[],"parallel_tool_calls":true,"previous_response_id":null,"prompt_cache_key":null,"reasoning":{"effort":"low","summary":null},"safety_identifier":null,"service_tier":"auto","store":true,"temperature":1.0,"text":{"format":{"type":"text"},"verbosity":"medium"},"tool_choice":"auto","tools":[],"top_logprobs":0,"top_p":1.0,"truncation":"disabled","usage":null,"user":null,"metadata":{}}}

event: response.in_progress
data: {"type":"response.in_progress","sequence_number":1,"response":{"id":"resp_689a2fd015448191a804ad63fdb613f60e5696160b26e781","object":"response","created_at":1754935248,"status":"in_progress","background":false,"error":null,"incomplete_details":null,"instructions":null,"max_output_tokens":4096,"max_tool_calls":null,"model":"gpt-5-2025-08-07","output":[],"parallel_tool_calls":true,"previous_response_id":null,"prompt_cache_key":null,"reasoning":{"effort":"low","summary":null},"safety_identifier":null,"service_tier":"auto","store":true,"temperature":1.0,"text":{"format":{"type":"text"},"verbosity":"medium"},"tool_choice":"auto","tools":[],"top_logprobs":0,"top_p":1.0,"truncation":"disabled","usage":null,"user":null,"metadata":{}}}

event: response.output_item.added
data: {"type":"response.output_item.added","sequence_number":2,"output_index":0,"item":{"id":"rs_689a2fd0b7b08191823b9d3993c07fc70e5696160b26e781","type":"reasoning","summary":[]}}

event: response.output_item.done
data: {"type":"response.output_item.done","sequence_number":3,"output_index":0,"item":{"id":"rs_689a2fd0b7b08191823b9d3993c07fc70e5696160b26e781","type":"reasoning","summary":[]}}

event: response.output_item.added
data: {"type":"response.output_item.added","sequence_number":4,"output_index":1,"item":{"id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","type":"message","status":"in_progress","content":[],"role":"assistant"}}

event: response.content_part.added
data: {"type":"response.content_part.added","sequence_number":5,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"part":{"type":"output_text","annotations":[],"logprobs":[],"text":""}}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":6,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":"Amsterdam","logprobs":[],"obfuscation":"KLcTNXt"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":7,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" is","logprobs":[],"obfuscation":"ZqWL13Sb7YycU"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":8,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" the","logprobs":[],"obfuscation":"ChCGlzWA7n96"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":9,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" constitutional","logprobs":[],"obfuscation":"T"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":10,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" capital","logprobs":[],"obfuscation":"cCxuXtY2"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":11,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" of","logprobs":[],"obfuscation":"rUBD4vWK12Moj"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":12,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" the","logprobs":[],"obfuscation":"BezXmbbfTGzM"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":13,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" Netherlands","logprobs":[],"obfuscation":"v04l"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":14,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":".","logprobs":[],"obfuscation":"NNXZQaJgOkSBnRA"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":15,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" The","logprobs":[],"obfuscation":"w3CrBMX5NSZp"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":16,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" seat","logprobs":[],"obfuscation":"9shtEnQK0Fb"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":17,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" of","logprobs":[],"obfuscation":"RhWVzGpqrOone"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":18,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" government","logprobs":[],"obfuscation":"5KNhW"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":19,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" and","logprobs":[],"obfuscation":"OXZqa4JwGpz5"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":20,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" the","logprobs":[],"obfuscation":"Ym6dFwaThhzj"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":21,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" royal","logprobs":[],"obfuscation":"qdywNFSbae"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":22,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" residence","logprobs":[],"obfuscation":"zgJf3g"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":23,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" are","logprobs":[],"obfuscation":"tjjlVGiyrE11"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":24,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" in","logprobs":[],"obfuscation":"PJ5lZA5UfIwXh"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":25,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" The","logprobs":[],"obfuscation":"FeCDvva7NuHL"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":26,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":" Hague","logprobs":[],"obfuscation":"G5gqRw5NVG"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","sequence_number":27,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"delta":".","logprobs":[],"obfuscation":"5Oko165nDgjrGWK"}

event: response.output_text.done
data: {"type":"response.output_text.done","sequence_number":28,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"text":"Amsterdam is the constitutional capital of the Netherlands. The seat of government and the royal residence are in The Hague.","logprobs":[]}

event: response.content_part.done
data: {"type":"response.content_part.done","sequence_number":29,"item_id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","output_index":1,"content_index":0,"part":{"type":"output_text","annotations":[],"logprobs":[],"text":"Amsterdam is the constitutional capital of the Netherlands. The seat of government and the royal residence are in The Hague."}}

event: response.output_item.done
data: {"type":"response.output_item.done","sequence_number":30,"output_index":1,"item":{"id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","type":"message","status":"completed","content":[{"type":"output_text","annotations":[],"logprobs":[],"text":"Amsterdam is the constitutional capital of the Netherlands. The seat of government and the royal residence are in The Hague."}],"role":"assistant"}}

event: response.completed
data: {"type":"response.completed","sequence_number":31,"response":{"id":"resp_689a2fd015448191a804ad63fdb613f60e5696160b26e781","object":"response","created_at":1754935248,"status":"completed","background":false,"error":null,"incomplete_details":null,"instructions":null,"max_output_tokens":4096,"max_tool_calls":null,"model":"gpt-5-2025-08-07","output":[{"id":"rs_689a2fd0b7b08191823b9d3993c07fc70e5696160b26e781","type":"reasoning","summary":[]},{"id":"msg_689a2fd133d081919b2ff584122ab87a0e5696160b26e781","type":"message","status":"completed","content":[{"type":"output_text","annotations":[],"logprobs":[],"text":"Amsterdam is the constitutional capital of the Netherlands. The seat of government and the royal residence are in The Hague."}],"role":"assistant"}],"parallel_tool_calls":true,"previous_response_id":null,"prompt_cache_key":null,"reasoning":{"effort":"low","summary":null},"safety_identifier":null,"service_tier":"auto","store":true,"temperature":1.0,"text":{"format":{"type":"text"},"verbosity":"medium"},"tool_choice":"auto","tools":[],"top_logprobs":0,"top_p":1.0,"truncation":"disabled","usage":{"input_tokens":24,"input_tokens_details":{"cached_tokens":0},"output_tokens":28,"output_tokens_details":{"reasoning_tokens":0},"total_tokens":52},"user":null,"metadata":{}}}
```

### Web Search

Request: 
```shell
curl "https://api.openai.com/v1/responses" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "gpt-4.1",
    "tools": [
      {
        "type": "web_search",
        "search_context_size": "low"
      }
    ],
    "input": "What is the weather like in Red Hook, Brooklyn"
  }'
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