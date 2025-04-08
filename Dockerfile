FROM golang:1.24.1 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY assets/ /build/assets/
COPY internal/ /build/internal/
COPY main.go /build/main.go

RUN CGO_ENABLED=0 GOOS=linux go build -o inu-desktop .

# ---

FROM archlinux:latest

RUN pacman -Syu --noconfirm

RUN pacman -S --noconfirm \
# needed to run inu
ffmpeg xfce4 xorg-server-xvfb dbus pulseaudio \
# extras
mpv bash sudo yt-dlp firefox && \
# clean up
rm -rf /var/cache/pacman

RUN \
mkdir /run/dbus/ && \
useradd -u 1000 -m -s /bin/bash inu

COPY --from=builder /build/inu-desktop /usr/bin/inu-desktop

ENV IN_CONTAINER=1

CMD ["/usr/bin/inu-desktop"]

