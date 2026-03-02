package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/fibonacci", handleUserFibonacci)

	addr := ":8080"
	fmt.Println("User Server listening on", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		panic(err)
	}
}

func handleUserFibonacci(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	hostname := strings.TrimSpace(q.Get("hostname"))
	fsPortStr := strings.TrimSpace(q.Get("fs_port"))
	numberStr := strings.TrimSpace(q.Get("number"))
	asIP := strings.TrimSpace(q.Get("as_ip"))
	asPortStr := strings.TrimSpace(q.Get("as_port"))

	if hostname == "" || fsPortStr == "" || numberStr == "" || asIP == "" || asPortStr == "" {
		http.Error(w, "missing query params: hostname, fs_port, number, as_ip, as_port", http.StatusBadRequest)
		return
	}

	fsPort, err := strconv.Atoi(fsPortStr)
	if err != nil || fsPort <= 0 || fsPort > 65535 {
		http.Error(w, "invalid fs_port", http.StatusBadRequest)
		return
	}

	asPort, err := strconv.Atoi(asPortStr)
	if err != nil || asPort <= 0 || asPort > 65535 {
		http.Error(w, "invalid as_port", http.StatusBadRequest)
		return
	}

	// number must be an integer (experiment requirement)
	if _, err := strconv.Atoi(numberStr); err != nil {
		http.Error(w, "number must be an integer", http.StatusBadRequest)
		return
	}

	// 1) ask AS: hostname -> VALUE(=ip/host)
	queryMsg := fmt.Sprintf("TYPE=A\nNAME=%s\n", hostname)
	asResp, err := udpSendAndRecv(asIP, asPort, queryMsg, 2*time.Second)
	if err != nil {
		http.Error(w, "failed to query AS: "+err.Error(), http.StatusBadGateway)
		return
	}

	value, err := parseValueFromAS(asResp)
	if err != nil {
		http.Error(w, "invalid AS response: "+err.Error(), http.StatusBadGateway)
		return
	}
	if value == "" {
		http.Error(w, "hostname not found in AS", http.StatusBadGateway)
		return
	}

	// 2) call FS
	fsURL := fmt.Sprintf("http://%s:%d/fibonacci?number=%s", value, fsPort, numberStr)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fsURL)
	if err != nil {
		http.Error(w, "failed to call FS: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// pass the status code of FS (more intuitive)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func parseValueFromAS(msg string) (string, error) {
	// AS returns format: TYPE=A\nNAME=...\nVALUE=...\nTTL=10\n
	// 注册返回 "Registered\n"，但 US 不会发注册，所以这里只解析 query response。
	lines := strings.Split(msg, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "VALUE=") {
			return strings.TrimPrefix(line, "VALUE="), nil
		}
	}
	// if there is no VALUE=, the format is wrong
	return "", errors.New("missing VALUE= in response")
}

func udpSendAndRecv(host string, port int, msg string, timeout time.Duration) (string, error) {
	raddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return "", err
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(timeout))

	if _, err := conn.Write([]byte(msg)); err != nil {
		return "", err
	}

	buf := make([]byte, 2048)
	n, err := conn.Read(buf)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}
