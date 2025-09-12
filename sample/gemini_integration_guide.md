# Gemini Integration Guide

This guide shows how to integrate and test the Modbus MCP Server with Google's Gemini AI.

## Prerequisites

### 1. Running Modbus MCP Server
```bash
# Build and start the server
cd /home/dev/workspace/go-mcp/sample
go build -o modbus-server main.go
./modbus-server --modbus-ip 192.168.1.22 --modbus-port 5002
```

### 2. Python Environment
```bash
# Install required packages
pip install requests google-generativeai
```

### 3. Gemini API Key
Get your API key from [Google AI Studio](https://makersuite.google.com/app/apikey)

## Integration Approaches

### Approach 1: Direct API Integration

```python
import google.generativeai as genai
import requests
import json

# Configure Gemini
genai.configure(api_key="YOUR_API_KEY")
model = genai.GenerativeModel('gemini-pro')

class ModbusMCPExtension:
    def __init__(self, mcp_url="http://localhost:8080/mcp"):
        self.mcp_url = mcp_url
        self.session = requests.Session()

    def call_mcp_tool(self, tool_name, arguments):
        """Call MCP tool and return result"""
        response = self.session.post(
            self.mcp_url,
            json={
                "jsonrpc": "2.0",
                "method": "tools/call",
                "params": {
                    "name": tool_name,
                    "arguments": arguments
                },
                "id": 1
            }
        )
        return response.json()

    def get_available_tools(self):
        """Get list of available MCP tools"""
        response = self.session.post(
            self.mcp_url,
            json={
                "jsonrpc": "2.0",
                "method": "tools/list",
                "id": 1
            }
        )
        result = response.json()
        return result["result"]["tools"]

# Initialize extension
mcp = ModbusMCPExtension()

# Example usage with Gemini
def modbus_assistant(user_query):
    # Get available tools
    tools = mcp.get_available_tools()

    # Create system prompt with tool information
    system_prompt = f"""
    You are a Modbus control assistant. You can interact with Modbus devices using these tools:

    {json.dumps(tools, indent=2)}

    When the user asks about Modbus operations, use the appropriate tools to:
    1. Read holding registers for analog values
    2. Read coils for digital inputs/outputs
    3. Write to holding registers to set analog values
    4. Write to coils to control digital outputs

    Always explain what you're doing and show the results clearly.
    """

    # Generate response with Gemini
    response = model.generate_content(f"{system_prompt}\n\nUser: {user_query}")

    # If Gemini suggests using a tool, execute it
    if "read-holding-registers" in response.text:
        result = mcp.call_mcp_tool("read-holding-registers", {"address": 0, "quantity": 10})
        return f"{response.text}\n\nResult: {json.dumps(result, indent=2)}"

    return response.text

# Test the integration
print(modbus_assistant("Read the first 10 holding registers"))
```

### Approach 2: Function Calling with Gemini

```python
import google.generativeai as genai
import requests
import json

genai.configure(api_key="YOUR_API_KEY")

# Define MCP tool functions for Gemini
modbus_tools = [
    {
        "name": "read_holding_registers",
        "description": "Read 16-bit integer values from Modbus holding registers",
        "parameters": {
            "type": "object",
            "properties": {
                "address": {
                    "type": "integer",
                    "description": "Starting address to read from (0-65535)"
                },
                "quantity": {
                    "type": "integer",
                    "description": "Number of registers to read (1-125)"
                }
            },
            "required": ["address", "quantity"]
        }
    },
    {
        "name": "read_coils",
        "description": "Read boolean values from Modbus coils",
        "parameters": {
            "type": "object",
            "properties": {
                "address": {
                    "type": "integer",
                    "description": "Starting address to read from (0-65535)"
                },
                "quantity": {
                    "type": "integer",
                    "description": "Number of coils to read (1-2000)"
                }
            },
            "required": ["address", "quantity"]
        }
    },
    {
        "name": "write_holding_registers",
        "description": "Write 16-bit integer values to Modbus holding registers",
        "parameters": {
            "type": "object",
            "properties": {
                "address": {
                    "type": "integer",
                    "description": "Starting address to write to (0-65535)"
                },
                "values": {
                    "type": "array",
                    "items": {"type": "integer"},
                    "description": "Array of uint16 values to write"
                }
            },
            "required": ["address", "values"]
        }
    },
    {
        "name": "write_coils",
        "description": "Write boolean values to Modbus coils",
        "parameters": {
            "type": "object",
            "properties": {
                "address": {
                    "type": "integer",
                    "description": "Starting address to write to (0-65535)"
                },
                "values": {
                    "type": "array",
                    "items": {"type": "boolean"},
                    "description": "Array of boolean values to write"
                }
            },
            "required": ["address", "values"]
        }
    }
]

class MCPToolExecutor:
    def __init__(self, mcp_url="http://localhost:8080/mcp"):
        self.mcp_url = mcp_url
        self.session = requests.Session()

    def execute_function(self, function_name, parameters):
        """Execute MCP tool based on Gemini function call"""

        # Map Gemini function names to MCP tool names
        tool_mapping = {
            "read_holding_registers": "read-holding-registers",
            "read_coils": "read-coils",
            "write_holding_registers": "write-holding-registers",
            "write_coils": "write-coils"
        }

        mcp_tool_name = tool_mapping.get(function_name)
        if not mcp_tool_name:
            return {"error": f"Unknown function: {function_name}"}

        # Call MCP tool
        response = self.session.post(
            self.mcp_url,
            json={
                "jsonrpc": "2.0",
                "method": "tools/call",
                "params": {
                    "name": mcp_tool_name,
                    "arguments": parameters
                },
                "id": 1
            }
        )

        return response.json()

# Initialize
executor = MCPToolExecutor()
model = genai.GenerativeModel('gemini-pro', tools=modbus_tools)

def chat_with_modbus_assistant():
    print("🤖 Modbus Assistant with Gemini")
    print("Type 'quit' to exit")
    print("-" * 50)

    chat = model.start_chat()

    while True:
        user_input = input("\nYou: ")
        if user_input.lower() == 'quit':
            break

        try:
            response = chat.send_message(user_input)

            # Handle function calls
            if hasattr(response, 'function_calls'):
                for function_call in response.function_calls:
                    print(f"\n🔧 Executing: {function_call.name}")
                    print(f"Parameters: {function_call.args}")

                    # Execute the function
                    result = executor.execute_function(
                        function_call.name,
                        dict(function_call.args)
                    )

                    print(f"Result: {json.dumps(result, indent=2)}")

                    # Send function result back to Gemini
                    response = chat.send_message(
                        genai.protos.Content(
                            parts=[genai.protos.Part(
                                function_response=genai.protos.FunctionResponse(
                                    name=function_call.name,
                                    response=result
                                )
                            )]
                        )
                    )

            print(f"\n🤖 Assistant: {response.text}")

        except Exception as e:
            print(f"❌ Error: {e}")

if __name__ == "__main__":
    chat_with_modbus_assistant()
```

### Approach 3: Custom MCP Bridge

```python
#!/usr/bin/env python3
"""
MCP to Gemini Bridge
Bridges MCP protocol to Gemini function calling
"""

import asyncio
import json
import websockets
import requests
from typing import Dict, Any
import google.generativeai as genai

class MCPGeminiBridge:
    def __init__(self, mcp_url="http://localhost:8080/mcp", gemini_api_key=None):
        self.mcp_url = mcp_url
        self.session = requests.Session()
        if gemini_api_key:
            genai.configure(api_key=gemini_api_key)
        self.model = genai.GenerativeModel('gemini-pro')

    async def handle_gemini_request(self, websocket, path):
        """Handle WebSocket connections from Gemini"""
        try:
            async for message in websocket:
                data = json.loads(message)

                if data.get("type") == "function_call":
                    # Execute MCP function
                    result = self.execute_mcp_function(
                        data["function_name"],
                        data["parameters"]
                    )

                    # Send result back
                    response = {
                        "type": "function_result",
                        "function_name": data["function_name"],
                        "result": result
                    }

                    await websocket.send(json.dumps(response))

        except Exception as e:
            print(f"WebSocket error: {e}")

    def execute_mcp_function(self, function_name, parameters):
        """Execute MCP function and return result"""

        # Map function names
        tool_mapping = {
            "read_holding_registers": "read-holding-registers",
            "read_coils": "read-coils",
            "write_holding_registers": "write-holding-registers",
            "write_coils": "write-coils"
        }

        mcp_tool_name = tool_mapping.get(function_name)
        if not mcp_tool_name:
            return {"error": f"Unknown function: {function_name}"}

        try:
            response = self.session.post(
                self.mcp_url,
                json={
                    "jsonrpc": "2.0",
                    "method": "tools/call",
                    "params": {
                        "name": mcp_tool_name,
                        "arguments": parameters
                    },
                    "id": 1
                }
            )

            if response.status_code == 200:
                return response.json()
            else:
                return {"error": f"HTTP {response.status_code}"}

        except Exception as e:
            return {"error": str(e)}

    def start_bridge(self, host="localhost", port=8765):
        """Start the MCP-Gemini bridge"""
        print(f"🚀 Starting MCP-Gemini Bridge on ws://{host}:{port}")

        start_server = websockets.serve(
            self.handle_gemini_request,
            host,
            port
        )

        asyncio.get_event_loop().run_until_complete(start_server)
        asyncio.get_event_loop().run_forever()

# Usage
if __name__ == "__main__":
    bridge = MCPGeminiBridge(gemini_api_key="YOUR_API_KEY")
    bridge.start_bridge()
```

## Testing Examples

### 1. Basic Connectivity Test

```python
# test_basic.py
import requests

def test_mcp_server():
    response = requests.post(
        "http://localhost:8080/mcp",
        json={
            "jsonrpc": "2.0",
            "method": "tools/list",
            "id": 1
        }
    )

    print("MCP Server Status:", response.status_code)
    print("Available Tools:", response.json())

if __name__ == "__main__":
    test_mcp_server()
```

### 2. Read Operations Test

```python
# test_reads.py
import requests

def test_read_operations():
    # Test reading holding registers
    response = requests.post(
        "http://localhost:8080/mcp",
        json={
            "jsonrpc": "2.0",
            "method": "tools/call",
            "params": {
                "name": "read-holding-registers",
                "arguments": {"address": 0, "quantity": 5}
            },
            "id": 1
        }
    )

    print("Holding Registers:", response.json())

    # Test reading coils
    response = requests.post(
        "http://localhost:8080/mcp",
        json={
            "jsonrpc": "2.0",
            "method": "tools/call",
            "params": {
                "name": "read-coils",
                "arguments": {"address": 0, "quantity": 8}
            },
            "id": 2
        }
    )

    print("Coils:", response.json())

if __name__ == "__main__":
    test_read_operations()
```

### 3. Write Operations Test

```python
# test_writes.py
import requests

def test_write_operations():
    # Test writing holding registers
    response = requests.post(
        "http://localhost:8080/mcp",
        json={
            "jsonrpc": "2.0",
            "method": "tools/call",
            "params": {
                "name": "write-holding-registers",
                "arguments": {
                    "address": 100,
                    "values": [1111, 2222, 3333]
                }
            },
            "id": 1
        }
    )

    print("Write Holding Registers:", response.json())

    # Test writing coils
    response = requests.post(
        "http://localhost:8080/mcp",
        json={
            "jsonrpc": "2.0",
            "method": "tools/call",
            "params": {
                "name": "write-coils",
                "arguments": {
                    "address": 50,
                    "values": [True, False, True, False, True]
                }
            },
            "id": 2
        }
    )

    print("Write Coils:", response.json())

if __name__ == "__main__":
    test_write_operations()
```

## Running the Tests

### 1. Start the MCP Server
```bash
cd /home/dev/workspace/go-mcp/sample
./modbus-server --modbus-ip 192.168.1.22 --modbus-port 5002
```

### 2. Run Basic Tests
```bash
python test_gemini.py
```

### 3. Test Individual Operations
```bash
python test_reads.py
python test_writes.py
```

### 4. Run Gemini Integration
```bash
# Set your API key
export GEMINI_API_KEY="your-api-key-here"

# Run the function calling example
python gemini_function_calling.py
```

## Troubleshooting

### Common Issues

1. **MCP Server Not Running**
   ```bash
   # Check if server is running
   curl http://localhost:8080/mcp

   # Start server if needed
   ./modbus-server --modbus-ip 192.168.1.22 --modbus-port 5002
   ```

2. **Modbus Connection Issues**
   ```bash
   # Test Modbus connectivity
   telnet 192.168.1.22 5002

   # Check server logs for connection errors
   ```

3. **Gemini API Issues**
   ```bash
   # Verify API key
   export GEMINI_API_KEY="your-correct-api-key"

   # Check API quota and limits
   ```

4. **Python Dependencies**
   ```bash
   pip install --upgrade google-generativeai requests
   ```

### Debug Mode

Enable debug logging in the MCP server:
```bash
# Server logs will show detailed information
./modbus-server --modbus-ip 192.168.1.22 --modbus-port 5002
```

## Next Steps

1. **Customize for Your Use Case**
   - Modify tool parameters based on your Modbus device
   - Add error handling specific to your setup
   - Implement authentication if needed

2. **Production Deployment**
   - Use the deployment guide in `docs/deployment.md`
   - Set up monitoring and logging
   - Configure backup and recovery

3. **Advanced Features**
   - Add batch operations for multiple devices
   - Implement caching for frequently read values
   - Add real-time monitoring capabilities

## Support

- 📖 **Documentation**: Check `docs/` directory
- 🐛 **Issues**: Report bugs on GitHub
- 💬 **Discussions**: Join community discussions
- 📧 **Contact**: Use GitHub issues for support

---

**Happy integrating! 🤖⚡**