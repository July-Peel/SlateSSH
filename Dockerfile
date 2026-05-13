# build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git build-base

COPY backend ./backend
COPY frontend ./frontend

WORKDIR /app/backend

RUN go mod tidy
RUN go build -o /app/slatessh ./cmd/slatessh

# runtime stage
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/slatessh /app/slatessh
COPY --from=builder /app/frontend /app/frontend

RUN mkdir -p /app/data

ENV HOST=0.0.0.0
ENV PORT=3210
ENV STACK_GO_DATA_DIR=/app/data

EXPOSE 3210

CMD ["/app/slatessh"]
