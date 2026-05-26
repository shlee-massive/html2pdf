from flask import Flask, request, Response
from weasyprint import HTML

app = Flask(__name__)


@app.post("/pdf")
def to_pdf():
    html = request.get_data(as_text=True)
    if not html:
        return Response("empty body", status=400)
    pdf = HTML(string=html, base_url=request.host_url).write_pdf()
    return Response(pdf, mimetype="application/pdf")


@app.get("/health")
def health():
    return {"ok": True}


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)
