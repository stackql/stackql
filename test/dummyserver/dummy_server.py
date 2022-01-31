# Python 3 server example
from http.server import BaseHTTPRequestHandler, HTTPServer
import os
import time

hostName = "localhost"
serverPort = 8080

TEST_BASE_DIR = os.path.abspath(
    os.path.join(
        os.path.dirname(os.path.abspath(__file__)),
        '..'
    )
)

class MyServer(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/google':
            return self.get_compute_disks()
        self.send_response(200)
        self.send_header("Content-type", "text/html")
        self.end_headers()
        self.wfile.write(bytes("<html><head><title>https://pythonbasics.org</title></head>", "utf-8"))
        self.wfile.write(bytes("<p>Request: %s</p>" % self.path, "utf-8"))
        self.wfile.write(bytes("<body>", "utf-8"))
        self.wfile.write(bytes("<p>This is an example web server.</p>", "utf-8"))
        self.wfile.write(bytes("</body></html>", "utf-8"))

    def get_compute_disks(self):
        with open(os.path.join(TEST_BASE_DIR, 'assets/response/google/compute/disks/disks-list.json'), 'rb') as f:
            bt = f.read()
        self.send_response(200)
        self.send_header("Content-type", "application/json")
        self.end_headers()
        self.wfile.write(bt)

if __name__ == "__main__":        
    webServer = HTTPServer((hostName, serverPort), MyServer)
    print("Server started http://%s:%s" % (hostName, serverPort))

    try:
        webServer.serve_forever()
    except KeyboardInterrupt:
        pass

    webServer.server_close()
    print("Server stopped.")
