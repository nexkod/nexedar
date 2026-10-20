package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port              string
	DataDir           string
	PublicURL         string
	AdminPassword     string
	APISecret         string
	MaxBuildsPerApp   int
	DeviceFarmURL     string
	DeviceFarmToken   string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	publicURL := os.Getenv("PUBLIC_URL")
	if publicURL == "" {
		publicURL = "http://localhost:" + port
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin"
	}

	apiSecret := os.Getenv("API_SECRET")
	if apiSecret == "" {
		apiSecret = "nexedar-default-secret"
	}

	maxBuilds := 3
	if val := os.Getenv("MAX_BUILDS_PER_APP"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			maxBuilds = parsed
		}
	}

	deviceFarmURL := os.Getenv("DEVICE_FARM_URL")
	if deviceFarmURL == "" {
		deviceFarmURL = os.Getenv("NEXAWAN_AGENT_URL")
	}

	deviceFarmToken := os.Getenv("DEVICE_FARM_TOKEN")
	if deviceFarmToken == "" {
		deviceFarmToken = os.Getenv("NEXAWAN_AUTH_TOKEN")
	}

	return &Config{
		Port:            port,
		DataDir:         dataDir,
		PublicURL:       publicURL,
		AdminPassword:   adminPassword,
		APISecret:       apiSecret,
		MaxBuildsPerApp: maxBuilds,
		DeviceFarmURL:   deviceFarmURL,
		DeviceFarmToken: deviceFarmToken,
	}
}
