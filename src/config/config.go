package config

import (
	"os"
	"strconv"
)

var (
	WEB_PORT, _ = strconv.Atoi(getEnv("WEB_PORT", "4845"))
	UDP_PORT, _ = strconv.Atoi(getEnv("UDP_PORT", "4845"))

	PUBLIC_IP = getEnv("PUBLIC_IP", "")

	IN_CONTAINER = envExists("IN_CONTAINER")

	SCREEN_WIDTH, _  = strconv.Atoi(getEnv("SCREEN_WIDTH", "1920"))
	SCREEN_HEIGHT, _ = strconv.Atoi(getEnv("SCREEN_HEIGHT", "1080"))
	FRAMERATE, _     = strconv.Atoi(getEnv("FRAMERATE", "60"))

	USE_NVIDIA = envExists("USE_NVIDIA")

	SUPERVISOR_LOGS = envExists("SUPERVISOR_LOGS")
)

func getEnv(key string, fallback string) string {
	value, exists := os.LookupEnv(key)
	if exists {
		return value
	} else {
		return fallback
	}
}

func envExists(key string) bool {
	_, exists := os.LookupEnv(key)
	return exists
}
