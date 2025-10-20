package internal

import "strconv"

var (
	WEB_PORT, _ = strconv.Atoi(getEnv("WEB_PORT", "4845"))
	UDP_PORT, _ = strconv.Atoi(getEnv("UDP_PORT", "4845"))

	PUBLIC_IP = getEnv("PUBLIC_IP", "")

	IN_CONTAINER = envExists("IN_CONTAINER")

	SCREEN_WIDTH, _  = strconv.Atoi(getEnv("SCREEN_WIDTH", "1920"))
	SCREEN_HEIGHT, _ = strconv.Atoi(getEnv("SCREEN_HEIGHT", "1080"))
	FRAMERATE, _     = strconv.Atoi(getEnv("FRAMERATE", "60"))

	SUPERVISOR_LOGS = envExists("SUPERVISOR_LOGS")
)
