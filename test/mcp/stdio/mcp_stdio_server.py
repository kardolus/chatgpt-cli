from fastmcp import FastMCP

mcp = FastMCP("demo-stdio")

@mcp.tool()
def echo(payload: dict) -> dict:
    return {"received": payload}

if __name__ == "__main__":
    # This runs an MCP server over stdio (stdin/stdout).
    mcp.run()

