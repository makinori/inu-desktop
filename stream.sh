#!/bin/bash

# gst-launch-1.0 videotestsrc ! videoconvert ! autovideosink

# gst-launch-1.0 videotestsrc ! x264enc ! \
# h264parse ! mp4mux ! filesink location=vid2.mp4

# GST_DEBUG=whipsink:5 gst-launch-1.0 videotestsrc ! videoconvert ! openh264enc ! rtph264pay ! \
# 'application/x-rtp,media=video,encoding-name=H264,payload=96,clock-rate=90000' ! \
# whip.sink_0 audiotestsrc ! audioconvert ! opusenc ! rtpopuspay ! \
# 'application/x-rtp,media=audio,-encodingname=opus,payload=111,clock-rate=48000,encoding-params=(string)2' ! \
# whipclientsink signaller::whip-endpoint=http://localhost:4880/whip
# # whip.sink_1 whipsink name=whip whip-endpoint=http://127.0.0.1:4880/whip


# gst-launch-1.0 videotestsrc \
# ! video/x-raw,width=1920,height=1080,format=I420 \
# ! x264enc speed-preset=ultrafast bitrate=2000 \
# ! video/x-h264,profile=baseline \
# ! whipclientsink signaller::whip-endpoint=http://127.0.0.1:4880/whip

# gst-launch-1.0 videotestsrc \
# ! video/x-raw,width=1920,height=1080,format=I420 \
# ! nvh264enc bitrate=2000 rc-mode=cbr-ld-hq preset=low-latency-hq zerolatency=true \
# ! video/x-h264,profile=baseline \
# ! whipclientsink signaller::whip-endpoint=http://127.0.0.1:4880/whip

# gst-launch-1.0 \
# ximagesrc show-pointer=1 use-damage=0 ! \
# videoscale ! videorate ! videoconvert ! \
# video/x-raw,width=1920,height=1080,framerate=30/1 ! \
# nvh264enc bitrate=2000 ! \
# mp4mux ! filesink location=vid2.mp4

# rc-mode=cbr tune=ultra-low-latency multi-pass=two-pass-quarter preset=p4 zerolatency=true ! \

# video/x-h264,profile=baseline ! queue ! \
# whipclientsink signaller::whip-endpoint=http://127.0.0.1:4880/whip


#   pulsesrc device=0 volume=1 ! \
#     queue ! \
#       opusenc bitrate=48000 audio-type=generic bandwidth=fullband ! queue ! \
#         rtpopuspay pt=111 ! \
#         udpsink host=127.0.0.1 port=5002

# fd=0 path=0 
# gst-launch-1.0 path=42 pipewiresrc ! videoconvert ! autovideosink

# gst-launch-1.0 \
# ximagesrc show-pointer=1 use-damage=0 ! \
# videoscale ! videorate ! videoconvert ! \
# video/x-raw,width=1920,height=1080,framerate=60/1 ! queue ! \
# x264enc pass=cbr bitrate=8000 key-int-max=10 tune=zerolatency speed-preset=veryfast ! \
# video/x-h264,profile=baseline ! queue ! \
# whipclientsink signaller::whip-endpoint=http://127.0.0.1:4880/whip

# gst-launch-1.0 \
# ximagesrc show-pointer=1 use-damage=0 \
# ! videoscale ! videorate ! videoconvert \
# ! video/x-raw,width=1920,height=1080,format=I420 ! queue \
# ! x264enc speed-preset=ultrafast bitrate=2000 \
# ! video/x-h264,profile=baseline ! queue \
# ! whipclientsink signaller::whip-endpoint=http://127.0.0.1:4880/whip


# gst-launch-1.0 videotestsrc \
# ! video/x-raw,width=1920,height=1080,format=I420 \
# ! x264enc speed-preset=ultrafast bitrate=2000 \
# ! video/x-h264,profile=baseline \
# ! whipclientsink signaller::whip-endpoint=http://127.0.0.1:4880/whip

# ffmpeg -nostats \
# -video_size 1920x1080 -framerate 60 -f x11grab -i :0 \
# -an vid2.mp4
# # -an -f rtp rtp://127.0.0.1:5004?pkt_size=1316 \

# ffmpeg -re -i jhg7m7j9.mp4 \
# -c:v h264_nvenc -b:v 5000K -rc cbr_ld_hq -preset llhq -zerolatency 1 \
# -an -f rtp rtp://127.0.0.1:5004?pkt_size=1316

# -c:v h264_nvenc -b:v 5000K -rc cbr_ld_hq -preset llhq -zerolatency 1 \

# ffmpeg -f kmsgrab -i - -vf 'hwdownload,format=bgr0' output.mp4