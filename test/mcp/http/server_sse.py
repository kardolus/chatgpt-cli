from typing import Any, Dict
import time
from fastmcp import FastMCP

mcp = FastMCP("EchoSSEServer")

@mcp.tool
def echo(payload: Dict[str, Any]) -> Dict[str, Any]:
    # Slow it down a bit so it feels “streamy” even if it’s a single SSE event
    time.sleep(0.25)
    return {
        "content": [
            {"type": "text", "text": f"payload={payload!r}"}
        ]
    }

if __name__ == "__main__":
    # Exposes http://127.0.0.1:8000/mcp
    mcp.run(transport="http", host="127.0.0.1", port=8000)
