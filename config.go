package main

import "strconv"

var (
	WEB_PORT, _ = strconv.Atoi(getEnv("WEB_PORT", "4845"))
	UDP_PORT, _ = strconv.Atoi(getEnv("UDP_PORT", "4845"))

	PUBLIC_IP = getEnv("PUBLIC_IP", "")

	IN_CONTAINER = envExists("IN_CONTAINER")
)
