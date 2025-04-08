FROM golang:1.24.1 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o inu-desktop .

# ---

FROM archlinux:latest

RUN pacman -Syu --noconfirm

RUN pacman -S --noconfirm \
# needed to run inu
ffmpeg \
# extras
mpv bash

RUN useradd -u 1000 -m -s /bin/bash inu

USER inu

COPY --from=builder /build/inu-desktop /usr/bin/inu-desktop

CMD ["/usr/bin/inu-desktop"]

