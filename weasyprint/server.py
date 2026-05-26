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
    # WeasyPrint 는 thread-safe 하지 않다 (공식 문서). werkzeug 개발 서버는
    # 기본이 threaded=True 라 동시 요청 시 같은 Pango/HarfBuzz 객체에 여러
    # 스레드가 접근 → use-after-free / SIGSEGV. 명시적으로 비활성.
    app.run(host="0.0.0.0", port=5000, threaded=False)
