package main

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
)

func closePort(port int) error {
	switch runtime.GOOS {
	case "windows":
		closePortWindows(port)
	case "darwin":
		closePortMac(port)
	default:
		fmt.Printf("Unsupported operating system: %s\n", runtime.GOOS)
	}
	return nil
}

func closePortWindows(port int) {
	cmd := exec.Command("cmd", "/C", "netstat -aon | findstr :"+strconv.Itoa(port))
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error checking port %d: %v\n", port, err)
		return
	}

	if len(output) > 0 {
		lines := string(output)
		var pid string
		fmt.Sscanf(lines, "%*s %*s %*s %*s %s", &pid)

		if pid != "" {
			cmd = exec.Command("taskkill", "/PID", pid, "/F")
			err = cmd.Run()
			if err != nil {
				fmt.Printf("Error killing process on port %d: %v\n", port, err)
			} else {
				fmt.Printf("Successfully closed port %d\n", port)
			}
		}
	}
}

func closePortMac(port int) {
	cmd := exec.Command("lsof", "-i", ":"+strconv.Itoa(port))
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error checking port %d: %v\n", port, err)
		return
	}

	if len(output) > 0 {
		lines := string(output)
		var pid string
		_, err = fmt.Sscanf(lines, "%*s %s", &pid)
		if err == nil && pid != "PID" {
			cmd = exec.Command("kill", "-9", pid)
			err = cmd.Run()
			if err != nil {
				fmt.Printf("Error killing process on port %d: %v\n", port, err)
			} else {
				fmt.Printf("Successfully closed port %d\n", port)
			}
		}
	}
}

func isPortInUse(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return true
	}
	ln.Close()
	return false
}
