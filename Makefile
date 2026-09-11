.PHONY: up down down-v

up:
	docker compose up --build

down:
	docker compose down

down-v:
	docker compose down -v

.PHONY: test-up test-down test-identity test-restaurant test-notification test-search test-order test-payment test

test-up:
	docker compose -f compose.test.yaml --profile test up -d

test-down:
	docker compose -f compose.test.yaml down -v

test-identity:
	docker compose -f compose.test.yaml exec -T identity-test sh -c "cd /app && go test -p 1 -count=1 ./tests/..."

test-restaurant:
	docker compose -f compose.test.yaml exec -T restaurant-test sh -c "cd /app && go test -p 1 -count=1 ./tests/..."

test-notification:
	cd notification-service && go test ./tests/...

test-search:
	docker compose -f compose.test.yaml exec -T search-test sh -c "cd /app && go test -p 1 -count=1 ./tests/..."

test-order:
	docker compose -f compose.test.yaml exec -T order-test sh -c "cd /app && go test -p 1 -count=1 ./tests/..."

test-payment:
	docker compose -f compose.test.yaml exec -T payment-test sh -c "cd /app && go test -p 1 -count=1 ./tests/..."

test: test-up test-identity test-restaurant test-notification test-search test-order test-payment

.PHONY: proto

proto:
	protoc \
		--go_out=payment-service --go_opt=module=payment-service \
		--go-grpc_out=payment-service --go-grpc_opt=module=payment-service \
		payment-service/api/proto/payment/v1/payment.proto

.PHONY: fmt vet lint

fmt:
	@for svc in identity-service restaurant-service notification-service search-service order-service payment-service; do \
		unformatted=$$(cd $$svc && gofmt -l .); \
		if [ -n "$$unformatted" ]; then \
			echo "$$unformatted"; \
			exit 1; \
		fi; \
	done

vet:
	@for svc in identity-service restaurant-service notification-service search-service order-service payment-service; do \
		(cd $$svc && go vet ./...) || exit 1; \
	done

lint: fmt vet
