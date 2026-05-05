package main

import (
	"fmt"
	"strconv"
	"strings"

	cratenet "github.com/aayushkdev/crate/internal/net"
)

func parsePublishedPorts(values []string) ([]cratenet.PublishedPort, error) {
	ports := make([]cratenet.PublishedPort, 0, len(values))
	for _, value := range values {
		port, err := parsePublishedPort(value)
		if err != nil {
			return nil, err
		}
		ports = append(ports, port)
	}

	if err := cratenet.ValidatePublishedPorts(ports); err != nil {
		return nil, err
	}

	return ports, nil
}

func parsePublishedPort(value string) (cratenet.PublishedPort, error) {
	protocol := "tcp"
	parts := strings.SplitN(value, "/", 2)
	if len(parts) == 2 {
		protocol = strings.ToLower(parts[1])
	}

	mapping := strings.Split(parts[0], ":")
	switch len(mapping) {
	case 1:
		port, err := strconv.Atoi(mapping[0])
		if err != nil {
			return cratenet.PublishedPort{}, fmt.Errorf("invalid published port %q", value)
		}
		return cratenet.PublishedPort{
			HostPort:      port,
			ContainerPort: port,
			Protocol:      protocol,
		}, nil
	case 2:
		hostPort, err := strconv.Atoi(mapping[0])
		if err != nil {
			return cratenet.PublishedPort{}, fmt.Errorf("invalid host port in %q", value)
		}
		containerPort, err := strconv.Atoi(mapping[1])
		if err != nil {
			return cratenet.PublishedPort{}, fmt.Errorf("invalid container port in %q", value)
		}
		return cratenet.PublishedPort{
			HostPort:      hostPort,
			ContainerPort: containerPort,
			Protocol:      protocol,
		}, nil
	default:
		return cratenet.PublishedPort{}, fmt.Errorf("invalid publish spec %q", value)
	}
}
