# ---------- build ----------
FROM golang:1.24-alpine AS build

WORKDIR /src

# คัดลอกไฟล์ dependency ก่อน เพื่อให้ layer นี้ถูกใช้ซ้ำเมื่อโค้ดเปลี่ยนแต่ dependency ไม่เปลี่ยน
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 ทำให้ได้ไบนารีที่ไม่ผูกกับ libc ของ base image
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api  ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# ---------- runtime ----------
FROM alpine:3.20

# ca-certificates สำหรับการเชื่อมต่อ TLS ขาออก, tzdata เพื่อให้เวลาไทยถูกต้องใน log
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 -h /app appuser

ENV TZ=Asia/Bangkok

WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /out/migrate /app/migrate

# โฟลเดอร์เก็บไฟล์ Excel ต้นฉบับ ถูก mount เป็น named volume ตอนรันจริง
RUN mkdir -p /app/storage && chown -R appuser:appuser /app

USER appuser
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1

CMD ["/app/api"]
