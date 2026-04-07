#!/usr/bin/env python3
import argparse
import json
import os
import signal
import shlex
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass
from pathlib import Path

import requests


REPO_ROOT = Path(__file__).resolve().parents[1]


@dataclass
class Check:
    name: str
    passed: bool
    detail: str = ""


class MCPClient:
    def __init__(self, endpoint: str):
        self.endpoint = endpoint
        self.session = requests.Session()
        self.req_id = 1

    def initialize(self):
        payload = {
            "jsonrpc": "2.0",
            "method": "initialize",
            "params": {
                "protocolVersion": "2024-11-05",
                "capabilities": {},
                "clientInfo": {"name": "happy-corner-test", "version": "1"},
            },
            "id": self._next_id(),
        }
        self.session.post(self.endpoint, json=payload, timeout=5).raise_for_status()

    def list_tools(self):
        payload = {"jsonrpc": "2.0", "method": "tools/list", "params": {}, "id": self._next_id()}
        resp = self.session.post(self.endpoint, json=payload, timeout=5)
        resp.raise_for_status()
        return resp.json()

    def call_tool(self, name: str, arguments: dict):
        payload = {
            "jsonrpc": "2.0",
            "method": "tools/call",
            "params": {"name": name, "arguments": arguments},
            "id": self._next_id(),
        }
        resp = self.session.post(self.endpoint, json=payload, timeout=8)
        resp.raise_for_status()
        body = resp.json()
        if "error" in body:
            return True, str(body["error"])
        result = body.get("result", {})
        is_error = bool(result.get("isError", False))
        text = ""
        for c in result.get("content", []):
            if c.get("type") == "text":
                text = c.get("text", "")
                break
        return is_error, text

    def _next_id(self):
        self.req_id += 1
        return self.req_id


def wait_health(base_url: str, timeout_s: float = 12.0):
    deadline = time.time() + timeout_s
    health = f"{base_url}/health"
    while time.time() < deadline:
        try:
            r = requests.get(health, timeout=0.4)
            if r.status_code == 200:
                return True
        except requests.RequestException:
            pass
        time.sleep(0.1)
    return False


def wait_port_free(host: str, port: int, timeout_s: float = 8.0):
    deadline = time.time() + timeout_s
    while time.time() < deadline:
        try:
            r = requests.get(f"http://{host}:{port}/health", timeout=0.2)
            if r.status_code:
                time.sleep(0.1)
                continue
        except requests.RequestException:
            return True
    return False


def run_checks(client: MCPClient, writes_expected_enabled: bool):
    checks = []
    expected_tools = {
        "read-holding-registers",
        "read-holding-registers-typed",
        "read-input-registers",
        "read-coils",
        "read-discrete-inputs",
        "write-holding-registers",
        "write-coils",
        "read-tag",
        "write-tag",
        "list-tags",
        "get-modbus-client-status",
    }

    tools_resp = client.list_tools()
    tools = {t.get("name") for t in tools_resp.get("result", {}).get("tools", [])}
    missing = sorted(expected_tools - tools)
    checks.append(Check("tools/list includes expected tools", len(missing) == 0, f"missing={missing}"))

    def expect_ok(name, args, must_contain=""):
        is_error, text = client.call_tool(name, args)
        ok = (not is_error) and (must_contain in text if must_contain else True)
        checks.append(Check(f"happy: {name} {args}", ok, text))

    def expect_err(name, args, must_contain=""):
        is_error, text = client.call_tool(name, args)
        ok = is_error and (must_contain in text if must_contain else True)
        checks.append(Check(f"corner: {name} {args}", ok, text))

    expect_ok("get-modbus-client-status", {}, '"driver"')
    expect_ok("read-input-registers", {"address": 0, "quantity": 2}, "Input registers")
    expect_ok("read-holding-registers", {"address": 0, "quantity": 2}, "Holding registers")
    expect_ok("read-coils", {"address": 0, "quantity": 4}, "Coils")
    expect_ok("read-discrete-inputs", {"address": 0, "quantity": 4}, "Discrete inputs")
    expect_ok("read-holding-registers-typed", {"address": 0, "data_type": "uint16"}, "decoded=")
    expect_ok("list-tags", {}, "setpoint")
    expect_ok("read-tag", {"name": "temp"}, '"decoded_value"')

    expect_err("read-input-registers", {"address": 0, "quantity": 0}, "quantity")
    expect_err("read-tag", {"name": "missing_tag"}, "not found")
    expect_err("write-tag", {"name": "temp", "holding_values": [5]}, "not writable")
    expect_err(
        "write-tag",
        {"name": "setpoint", "holding_values": [5], "numeric_value": 5},
        "ambiguous input",
    )
    expect_err(
        "write-tag",
        {"name": "relay", "coil_values": [True], "bool_value": True},
        "ambiguous input",
    )

    if writes_expected_enabled:
        expect_ok("write-holding-registers", {"address": 1, "values": [77]}, "Successfully wrote")
        expect_ok("write-coils", {"address": 1, "values": [True, False]}, "Successfully wrote")
        expect_ok("write-tag", {"name": "setpoint", "holding_values": [66]}, "written")
        expect_ok("write-tag", {"name": "relay", "bool_value": True}, "written")
        expect_err("write-holding-registers", {"address": 200, "values": [1]}, "guarded rejection")
    else:
        expect_err("write-holding-registers", {"address": 1, "values": [77]}, "guarded rejection")
        expect_err("write-tag", {"name": "setpoint", "holding_values": [66]}, "guarded rejection")

    return checks


def build_config(path: Path, writes_enabled: bool):
    text = f"""mock_mode: true
transport: streamable
write_policy:
  writes_enabled: {str(writes_enabled).lower()}
  holding_write_allowlist: \"0-20\"
  coil_write_allowlist: \"0-20\"
tags:
  - name: temp
    kind: holding_register
    address: 0
    quantity: 1
    access: read
    data_type: uint16
    byte_order: big
    word_order: msw
  - name: setpoint
    kind: holding_register
    address: 1
    quantity: 1
    access: read_write
    data_type: uint16
    byte_order: big
    word_order: msw
  - name: relay
    kind: coil
    address: 2
    quantity: 1
    access: read_write
"""
    path.write_text(text, encoding="utf-8")


def run_phase(base_cmd: str, config_path: Path, base_url: str, writes_enabled: bool):
    cmd = shlex.split(base_cmd) + ["--config", str(config_path), "--transport", "streamable"]
    proc = subprocess.Popen(
        cmd,
        cwd=str(REPO_ROOT),
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        start_new_session=True,
    )
    try:
        if not wait_health(base_url):
            return [Check(f"server startup ({'writes-on' if writes_enabled else 'writes-off'})", False, "health timeout")]
        client = MCPClient(f"{base_url}/mcp")
        client.initialize()
        return run_checks(client, writes_enabled)
    finally:
        try:
            os.killpg(proc.pid, signal.SIGTERM)
        except ProcessLookupError:
            pass
        try:
            proc.wait(timeout=2)
        except subprocess.TimeoutExpired:
            try:
                os.killpg(proc.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
        wait_port_free("127.0.0.1", 8080)


def main():
    parser = argparse.ArgumentParser(description="Happy-path + corner-case MCP tool test script")
    parser.add_argument("--base-cmd", default="go run .", help="Command used to start server")
    parser.add_argument("--base-url", default="http://127.0.0.1:8080", help="HTTP base URL")
    args = parser.parse_args()

    with tempfile.TemporaryDirectory(prefix="happy-corner-") as td:
        td_path = Path(td)
        cfg_on = td_path / "writes-on.yaml"
        cfg_off = td_path / "writes-off.yaml"
        build_config(cfg_on, writes_enabled=True)
        build_config(cfg_off, writes_enabled=False)

        checks = []
        checks.extend(run_phase(args.base_cmd, cfg_on, args.base_url, writes_enabled=True))
        checks.extend(run_phase(args.base_cmd, cfg_off, args.base_url, writes_enabled=False))

    failed = [c for c in checks if not c.passed]
    for c in checks:
        icon = "PASS" if c.passed else "FAIL"
        print(f"[{icon}] {c.name}")
        if c.detail and not c.passed:
            print(f"       {c.detail}")

    summary = {
        "total": len(checks),
        "passed": len(checks) - len(failed),
        "failed": len(failed),
    }
    print("\nSummary:", json.dumps(summary))
    if failed:
        sys.exit(1)


if __name__ == "__main__":
    main()
