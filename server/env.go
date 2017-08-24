package main

import "os"

func getIP() string {
	return ""
}

func getPort() string {
	s := os.Getenv("PORT")
	if s == "" {
		return "8080"
	}
	return s
}
