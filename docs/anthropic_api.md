# Anthropic API

## Documentation
https://docs.anthropic.com/en/api/messages
https://console.anthropic.com/settings/plans

## CURLS

Request:

```shell
curl https://api.anthropic.com/v1/messages \
     --header "x-api-key: $ANTHROPIC_API_KEY" \
     --header "anthropic-version: 2023-06-01" \
     --header "content-type: application/json" \
     --data \
'{
    "model": "claude-3-5-sonnet-20240620",
    "max_tokens": 1024,
    "messages": [
        {"role": "user", "content": "Hello, world"}
    ]
}' | jq .
```

Response:

```shell
{
  "id": "msg_012899dyMDyCX4FgMNNbao8k",
  "type": "message",
  "role": "assistant",
  "model": "claude-3-5-sonnet-20240620",
  "content": [
    {
      "type": "text",
      "text": "Hello! How can I assist you today? Feel free to ask me any questions or let me know if you need help with anything."
    }
  ],
  "stop_reason": "end_turn",
  "stop_sequence": null,
  "usage": {
    "input_tokens": 10,
    "output_tokens": 30
  }
}
```

Request (enabling stream):

```shell
curl https://api.anthropic.com/v1/messages \
     --header "x-api-key: $ANTHROPIC_API_KEY" \
     --header "anthropic-version: 2023-06-01" \
     --header "content-type: application/json" \
     --data \
'{
    "model": "claude-3-5-sonnet-20240620",
    "max_tokens": 1024,
    "stream": true,
    "messages": [
        {"role": "user", "content": "Hello, world"}
    ]
}' | jq .
```

Response:

```shell
event: message_start
data: {"type":"message_start","message":{"id":"msg_01RQD3yEtQ6obCh6K8aeFMQd","type":"message","role":"assistant","model":"claude-3-5-sonnet-20240620","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":3}}  }

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}     }

event: ping
data: {"type": "ping"}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello! How"}               }

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" can"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" I assist you today?"}        }

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" Feel"}    }

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" free to ask me any"}    }

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" questions or let"}            }

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" me know if you"}            }

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" nee"}  }

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"d help with anything."}   }

event: content_block_stop
data: {"type":"content_block_stop","index":0          }

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":30}       }

event: message_stop
data: {"type":"message_stop"
```