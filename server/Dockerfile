# Image เดียวบรรจุสองไบนารี: api (เซิร์ฟเวอร์) และ migrate (สร้างตาราง/บัญชี admin)
# ทำแบบนี้เพื่อให้เวอร์ชันของ schema กับโค้ดที่ใช้ schema นั้นออกจาก build เดียวกันเสมอ

# ---------- build ----------
# ต้องตรงกับบรรทัด go ใน go.mod (go 1.26.0) ไม่งั้น build ล้มตั้งแต่ go mod download
FROM golang:1.26-alpine AS build

WORKDIR /src

# คัดลอกไฟล์ dependency ก่อน เพื่อให้ layer นี้ถูกใช้ซ้ำเมื่อโค้ดเปลี่ยนแต่ dependency ไม่เปลี่ยน
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 ทำให้ได้ไบนารีที่ไม่ผูกกับ libc ของ base image
# -trimpath ตัด path ของเครื่อง build ออกจาก stack trace
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
# ต้องมีอยู่ใน image พร้อมเจ้าของที่ถูกต้อง เพราะ Docker คัดลอกสิทธิ์ของโฟลเดอร์นี้
# ไปตั้งให้ volume ที่สร้างใหม่ ถ้าไม่ทำ volume จะเป็นของ root แล้ว appuser เขียนไม่ได้
RUN mkdir -p /app/storage && chown -R appuser:appuser /app

USER appuser
EXPOSE 8080

# ใช้ /healthz (liveness) ไม่ใช่ /readyz เพราะ healthcheck นี้ตัดสินใจว่าจะ restart container หรือไม่
# ถ้าใช้ /readyz แล้วฐานข้อมูลล่ม api จะโดน restart วนไปเรื่อย ๆ ทั้งที่ตัวมันไม่ได้ผิด
# อ่าน $PORT ตอนรัน เพื่อให้ยังตรวจถูกตัวถ้าวันหลังเปลี่ยนพอร์ตภายใน container
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- "http://127.0.0.1:${PORT:-8080}/healthz" || exit 1

CMD ["/app/api"]
