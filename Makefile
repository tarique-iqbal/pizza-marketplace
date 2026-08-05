.PHONY: up down down-v test-up test-down test-identity test-restaurant test-email test fmt vet lint

up:
	docker compose up --build

down:
	docker compose down

down-v:
	docker compose down -v

test-up:
	docker compose -f compose.test.yaml --profile test up -d

test-down:
	docker compose -f compose.test.yaml down -v

test-identity:
	docker compose -f compose.test.yaml exec -T identity-test sh -c "cd /app && go test -p 1 -count=1 ./..."

test-restaurant:
	docker compose -f compose.test.yaml exec -T restaurant-test sh -c "cd /app && go test -p 1 -count=1 ./..."

test-email:
	cd email-service && go test ./...

test: test-up test-identity test-restaurant test-email

fmt:
	@for svc in identity-service restaurant-service email-service; do \
		unformatted=$$(cd $$svc && gofmt -l .); \
		if [ -n "$$unformatted" ]; then \
			echo "$$unformatted"; \
			exit 1; \
		fi; \
	done

vet:
	@for svc in identity-service restaurant-service email-service; do \
		(cd $$svc && go vet ./...) || exit 1; \
	done

lint: fmt vet
