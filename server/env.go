package main

import "os"

func getIP() string {
	return os.Getenv("OPENSHIFT_GO_IP")
}

func getPort() string {
	s := os.Getenv("OPENSHIFT_GO_PORT")
	if s == "" {
		return "8000"
	}
	return s
}
