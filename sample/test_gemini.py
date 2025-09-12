#!/usr/bin/env python3
"""
Gemini MCP Integration Test Script
Tests the Modbus MCP Server with Google's Gemini AI
"""

import requests
import json
import time
from typing import Dict, Any

class GeminiMCPTester:
    def __init__(self, mcp_url: str = "http://localhost:8080/mcp"):
        self.mcp_url = mcp_url
        self.session = requests.Session()

    def test_mcp_connection(self) -> bool:
        """Test if MCP server is running and accessible"""
        try:
            response = self.session.post(
                self.mcp_url,
                json={
                    "jsonrpc": "2.0",
                    "method": "tools/list",
                    "id": 1
                }
            )
            if response.status_code == 200:
                result = response.json()
                tools = result.get("result", {}).get("tools", [])
                print(f"✅ MCP Server connected! Found {len(tools)} tools:")
                for tool in tools:
                    print(f"  - {tool['name']}: {tool['description']}")
                return True
            else:
                print(f"❌ MCP Server error: {response.status_code}")
                return False
        except Exception as e:
            print(f"❌ Cannot connect to MCP server: {e}")
            return False

    def test_read_operations(self):
        """Test read operations through MCP"""
        print("\n🔍 Testing Read Operations:")

        # Test reading holding registers
        print("\n📊 Reading Holding Registers:")
        response = self.call_tool("read-holding-registers", {
            "address": 0,
            "quantity": 5
        })
        if response:
            print(f"Result: {response}")

        # Test reading coils
        print("\n🔌 Reading Coils:")
        response = self.call_tool("read-coils", {
            "address": 0,
            "quantity": 8
        })
        if response:
            print(f"Result: {response}")

    def test_write_operations(self):
        """Test write operations through MCP"""
        print("\n✏️ Testing Write Operations:")

        # Test writing holding registers
        print("\n📝 Writing Holding Registers:")
        response = self.call_tool("write-holding-registers", {
            "address": 100,
            "values": [1111, 2222, 3333]
        })
        if response:
            print(f"Result: {response}")

        # Test writing coils
        print("\n⚡ Writing Coils:")
        response = self.call_tool("write-coils", {
            "address": 50,
            "values": [True, False, True, False, True]
        })
        if response:
            print(f"Result: {response}")

    def call_tool(self, tool_name: str, arguments: Dict[str, Any]) -> str:
        """Call an MCP tool and return the result"""
        try:
            response = self.session.post(
                self.mcp_url,
                json={
                    "jsonrpc": "2.0",
                    "method": "tools/call",
                    "params": {
                        "name": tool_name,
                        "arguments": arguments
                    },
                    "id": int(time.time() * 1000)  # Unique ID
                }
            )

            if response.status_code == 200:
                result = response.json()
                if "error" in result:
                    return f"❌ Error: {result['error']}"
                elif "result" in result:
                    content = result["result"].get("content", [])
                    if content:
                        return content[0].get("text", "No text content")
                    else:
                        return "✅ Success (no content)"
                else:
                    return f"Unexpected response: {result}"
            else:
                return f"❌ HTTP {response.status_code}: {response.text}"

        except Exception as e:
            return f"❌ Exception: {e}"

def main():
    print("🤖 Gemini MCP Integration Test")
    print("=" * 50)

    tester = GeminiMCPTester()

    # Test MCP connection
    if not tester.test_mcp_connection():
        print("\n❌ Cannot proceed without MCP server connection")
        print("Make sure to start the Modbus MCP server first:")
        print("  cd /path/to/project/sample")
        print("  ./modbus-server --modbus-ip YOUR_MODBUS_IP --modbus-port YOUR_MODBUS_PORT")
        return

    # Test read operations
    tester.test_read_operations()

    # Test write operations
    tester.test_write_operations()

    print("\n" + "=" * 50)
    print("✅ MCP Integration Test Complete!")
    print("\n📝 Next Steps:")
    print("1. Use this script to verify your MCP server works")
    print("2. Integrate with Gemini using the API examples below")
    print("3. Test with real Modbus hardware")

if __name__ == "__main__":
    main()