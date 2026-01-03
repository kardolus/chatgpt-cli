from typing import Any, Dict
from fastmcp import FastMCP

mcp = FastMCP("EchoServer")

@mcp.tool
def echo(payload: Dict[str, Any]) -> Dict[str, Any]:
    return {
        "content": [
            {"type": "text", "text": str(payload)}
        ]
    }

if __name__ == "__main__":
    mcp.run(transport="http", host="127.0.0.1", port=8000)