# FROM golang:1.24.1-alpine AS builder
# RUN apk add --no-cache build-base libx11-dev

FROM archlinux:latest AS builder

RUN pacman -Syu --noconfirm base-devel go libx11 libxtst

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY main.go /build/main.go
COPY src/ /build/src/

# RUN CGO_ENABLED=0 GOOS=linux go build -o inu-desktop .

RUN GOOS=linux go build -o inu-desktop .

# ---

FROM archlinux:latest

RUN \
# update pacman conf
sed -i "s/#Color/Color/" /etc/pacman.conf && \
sed -i "s/#ParallelDownloads/ParallelDownloads/" /etc/pacman.conf && \
sed -i "s/^NoProgressBar/#NoProgressBar/" /etc/pacman.conf && \
echo "[multilib]" >> /etc/pacman.conf && \
echo "Include = /etc/pacman.d/mirrorlist" >> /etc/pacman.conf && \
# update packages and get reflector
pacman -Syu --noconfirm reflector && \
# get fastest
reflector --country US --latest 25 --score 25 --sort rate \
--protocol https --verbose --save /etc/pacman.d/mirrorlist && \
# clean up
rm -rf /var/cache/pacman 

RUN \
# newer mesa unfortunately breaks xvfb
curl -o mesa.tar.zst https://archive.archlinux.org/packages/m/mesa/mesa-1%3A23.3.1-1-x86_64.pkg.tar.zst && \
# debian 13 comes with nvidia 550 so we need to use that one
# downgrading here however breaks gstreamer, so just keep nvidia up to date 
# curl -o nvidia-utils.tar.zst https://archive.archlinux.org/packages/n/nvidia-utils/nvidia-utils-550.90.07-4-x86_64.pkg.tar.zst && \
# curl -o lib32-nvidia-utils.tar.zst https://archive.archlinux.org/packages/l/lib32-nvidia-utils/lib32-nvidia-utils-550.90.07-1-x86_64.pkg.tar.zst && \
# install above and clean up
pacman -U --noconfirm *.tar.zst && \
rm -f *.tar.zst && \
# needed to run inu
pacman -S --noconfirm \
gstreamer gst-plugins-base gst-plugins-good gst-plugins-bad gst-plugins-ugly \
xfce4 xorg-server-xvfb xclip dbus pulseaudio \
nvidia-utils lib32-nvidia-utils && \
# clean up
rm -rf /var/cache/pacman

RUN pacman -S --noconfirm \
# programs
mpv bash sudo yt-dlp firefox \
# fonts
ttf-cascadia-code \
# for building
git debugedit binutils fakeroot go make gcc patch && \
# clean up
rm -rf /var/cache/pacman

RUN \
mkdir /run/dbus/ && \
# add user
useradd -u 1000 -m -G wheel -s /bin/bash inu && \
passwd -d inu && \
sed -i "s/# %wheel ALL=(ALL:ALL) ALL/%wheel ALL=(ALL:ALL) ALL/" /etc/sudoers && \
# make necessary paths
mkdir -p /run/user/1000 && \
chown inu:inu /run/user/1000 && \
chmod 700 /run/user/1000 && \
mkdir -p /tmp/.X11-unix && \
chmod 1777 /tmp/.X11-unix && \
# generate locales and other
echo 'en_US.UTF-8 UTF-8' > /etc/locale.gen && locale-gen && \
dbus-uuidgen --ensure

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

ENV \
LANG=en_US.UTF-8 \
LANGUAGE=en_US:en \
LC_ALL=en_US.UTF-8 \
\
IN_CONTAINER=1 \
\
DISPLAY=:0 \
XDG_SESSION_TYPE=x11

CMD ["/usr/bin/inu-desktop"]

