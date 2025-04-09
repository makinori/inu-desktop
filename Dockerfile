# FROM golang:1.24.1-alpine AS builder
# RUN apk add --no-cache build-base libx11-dev

FROM archlinux:latest AS builder

RUN pacman -Syu --noconfirm base-devel go libx11 libxtst

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY assets/ /build/assets/
COPY internal/ /build/internal/
COPY main.go /build/main.go

# RUN CGO_ENABLED=0 GOOS=linux go build -o inu-desktop .

RUN GOOS=linux go build -o inu-desktop .

# ---

FROM archlinux:latest

RUN pacman -Syu --noconfirm

RUN pacman -S --noconfirm \
# needed to run inu
ffmpeg xfce4 xorg-server-xvfb dbus pulseaudio && \
# clean up
rm -rf /var/cache/pacman

RUN pacman -S --noconfirm \
# programs
mpv bash sudo yt-dlp firefox \
# fonts
ttf-cascadia-code \
# for building
git debugedit binutils fakeroot go make gcc patch \
# drivers
nvidia-utils && \
# clean up
rm -rf /var/cache/pacman

RUN \
mkdir /run/dbus/ && \
# add user
useradd -u 1000 -m -G wheel -s /bin/bash inu && \
passwd -d inu && \
sed -i "s/# %wheel ALL=(ALL:ALL) ALL/%wheel ALL=(ALL:ALL) ALL/" /etc/sudoers && \
# improve pacman
sed -i "s/#Color/Color/" /etc/pacman.conf && \
sed -i "s/#ParallelDownloads/ParallelDownloads/" /etc/pacman.conf && \
sed -i "s/^NoProgressBar/#NoProgressBar/" /etc/pacman.conf && \
echo "[multilib]" >> /etc/pacman.conf && \
echo "Include = /etc/pacman.d/mirrorlist" >> /etc/pacman.conf && \
pacman -Sy && \
# generate locales and other
echo 'en_US.UTF-8 UTF-8' > /etc/locale.gen && locale-gen && \
dbus-uuidgen --ensure

ENV \
LANG=en_US.UTF-8 \
LANGUAGE=en_US:en \
LC_ALL=en_US.UTF-8

RUN pacman -S --noconfirm \
# lib32 drivers
lib32-nvidia-utils && \
# clean up
rm -rf /var/cache/pacman

# get yay
RUN \
git clone https://aur.archlinux.org/yay.git /yay && \
chown -R inu:inu /yay && \
cd /yay && su inu -c "makepkg" && \
pacman -U --noconfirm *.tar.zst && \
cd .. && rm -rf /yay

# get from aur
RUN \
su inu -c "yay -S --noconfirm \
otf-sn-pro papirus-icon-theme ff2mpv-native-messaging-host-git" && \
rm -rf /home/inu/.cache

# install user settings
COPY user-settings.tar.gz /user-settings.tar.gz
RUN tar -C /home/inu -xf /user-settings.tar.gz && \
rm -f /user-settings.tar.gz

COPY --from=builder /build/inu-desktop /usr/bin/inu-desktop

ENV DISPLAY=:0
ENV XDG_SESSION_TYPE=x11

ENV IN_CONTAINER=1

CMD ["/usr/bin/inu-desktop"]

