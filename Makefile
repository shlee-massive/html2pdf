.PHONY: up down build run serve run-ko run-ja run-en clean health

up:
	docker compose up -d --build
	@echo ""
	@echo "  Gotenberg:   http://localhost:3000"
	@echo "  WeasyPrint:  http://localhost:5001/health"

down:
	docker compose down

build:
	go build -o bin/htp .

run:
	go run . -dump-html

# 브라우저 데모 서버: http://localhost:8080
serve:
	go run . -serve

run-ko:
	go run . -locale ko -dump-html

run-ja:
	go run . -locale ja -dump-html

run-en:
	go run . -locale en -dump-html

# 백엔드별 단독 실행
run-gotenberg:
	go run . -backend gotenberg

run-weasyprint:
	go run . -backend weasyprint

run-docraptor:
	go run . -backend docraptor

health:
	@echo "Gotenberg:"  && curl -sf http://localhost:3000/health  | head -c 200 ; echo
	@echo "WeasyPrint:" && curl -sf http://localhost:5001/health | head -c 200 ; echo

clean:
	rm -rf output/*.pdf output/*.html bin
