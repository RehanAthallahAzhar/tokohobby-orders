# --- Stage 1: Build ---
# Menggunakan image Go resmi sebagai basis untuk kompilasi
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy local dependencies (messaging and protos)
COPY messaging /app/messaging
COPY protos /app/protos

# Copy orders service
COPY orders /app/orders
WORKDIR /app/orders

# Download dependencies (will use local dependencies via replace directives)
RUN go mod download
# Build aplikasi
# CGO_ENABLED=0 untuk membuat binary yang statis (tidak tergantung library C)
# GOOS=linux karena kita akan menjalankannya di base image Alpine Linux
# -o /app/server akan menghasilkan output binary bernama 'server' di direktori /app
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/web/main.go


# --- Stage 2: Final Image ---
# Menggunakan base image yang sangat kecil (Alpine Linux) untuk produksi
FROM alpine:latest

WORKDIR /app

# Copy binary yang sudah di-build dari stage 'builder'
COPY --from=builder /app/server .

# (Opsional) Jika Anda punya file konfigurasi atau template yang perlu di-copy
# Contoh: COPY --from=builder /app/internal/configs/config.yaml .

# migration
COPY --from=builder /app/orders/db/migrations ./db/migrations

# Expose port yang digunakan oleh aplikasi Anda di dalam container
# Ganti 8080 jika aplikasi Anda berjalan di port lain
EXPOSE 8080

# Command untuk menjalankan aplikasi saat container dimulai
CMD ["./server"]