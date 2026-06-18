from flask import Flask, jsonify
import datetime as dt


app = Flask(__name__)

@app.route("/")
def welcome():
    return "Welcome to my homelab API!"

@app.route("/time")
def timeSinceStart():
    return jsonify({
        "time": dt.datetime.now().isoformat()
    })

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)


















