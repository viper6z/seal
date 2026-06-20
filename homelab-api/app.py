from flask import Flask, jsonify
import datetime as dt
import time
from datetime import timedelta

app = Flask(__name__)
START = time.monotonic()

@app.route("/")
def welcome():
    return "Welcome to my homelab API!"

@app.route("/time")
def uptime():
    seconds = time.monotonic() - START
    return jsonify({
        "uptime_seconds": round(seconds, 1),
        "uptime_human": str(timedelta(seconds=int(seconds)))
    })

@app.route("/health")
def health():
    return jsonify({
        "status" : "healthy"
    }), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)


















