package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type RegisterRequest struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
	ASIP     string `json:"as_ip"`
	ASPort   int    `json:"as_port"`
}

func main() {
	go autoRegister()
	mux := http.NewServeMux()
	mux.HandleFunc("/register", handleRegister)
	mux.HandleFunc("/fibonacci", handleFibonacci)

	addr := ":9090"
	fmt.Println("Fibonacci Server listening on", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		panic(err)
	}
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req RegisterRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	req.Hostname = strings.TrimSpace(req.Hostname)
	req.IP = strings.TrimSpace(req.IP)
	req.ASIP = strings.TrimSpace(req.ASIP)

	//  docker-compose
	// e.g. HOSTNAME=fibonacci.com FS_IP=fs AS_IP=as AS_PORT=53333
	if req.Hostname == "" {
		req.Hostname = strings.TrimSpace(os.Getenv("HOSTNAME"))
	}
	if req.IP == "" {
		req.IP = strings.TrimSpace(os.Getenv("FS_IP"))
	}
	if req.ASIP == "" {
		req.ASIP = strings.TrimSpace(os.Getenv("AS_IP"))
	}
	if req.ASPort == 0 {
		if p := strings.TrimSpace(os.Getenv("AS_PORT")); p != "" {
			if v, err := strconv.Atoi(p); err == nil {
				req.ASPort = v
			}
		}
	}

	if req.Hostname == "" || req.IP == "" || req.ASIP == "" || req.ASPort <= 0 || req.ASPort > 65535 {
		http.Error(w, "missing/invalid fields (hostname, ip, as_ip, as_port)", http.StatusBadRequest)
		return
	}

	// if req.ASIP and req.ASPort is valid, send the register message to AS
	msg := fmt.Sprintf("TYPE=A\nNAME=%s\nVALUE=%s\nTTL=10\n", req.Hostname, req.IP)

	resp, err := udpSendAndRecv(req.ASIP, req.ASPort, msg, 2*time.Second)
	if err != nil {
		http.Error(w, "failed to register with AS: "+err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	// AS returns "Registered\n"
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(strings.TrimSpace(resp) + "\n"))
}

func handleFibonacci(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query().Get("number")
	n, err := strconv.Atoi(q)
	if err != nil {
		http.Error(w, "number must be an integer", http.StatusBadRequest)
		return
	}
	if n < 0 {
		http.Error(w, "number must be non-negative", http.StatusBadRequest)
		return
	}

	// uint64 safe limit: fib(93) will overflow uint64, so limit to 92
	if n > 92 {
		http.Error(w, "number too large (max 92 for uint64)", http.StatusBadRequest)
		return
	}

	val := fibUint64(n)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(strconv.FormatUint(val, 10)))
}

func fibUint64(n int) uint64 {
	if n == 0 {
		return 0
	}
	if n == 1 {
		return 1
	}
	var a uint64 = 0
	var b uint64 = 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b
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
func autoRegister() {
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		hostname = "fibonacci.com"
	}

	ip := os.Getenv("FS_IP")
	if ip == "" {
		ip = "fs"
	}

	asIP := os.Getenv("AS_IP")
	if asIP == "" {
		asIP = "as"
	}

	asPort := 53333
	if asPortStr := os.Getenv("AS_PORT"); asPortStr != "" {
		if p, err := strconv.Atoi(asPortStr); err == nil {
			asPort = p
		}
	}

	msg := fmt.Sprintf("TYPE=A\nNAME=%s\nVALUE=%s\nTTL=10\n", hostname, ip)

	// ✅ 最多重试 15 次，每次间隔 1 秒
	for i := 1; i <= 15; i++ {
		_, err := udpSendAndRecv(asIP, asPort, msg, 2*time.Second)
		if err == nil {
			fmt.Println("Auto registered with AS")
			return
		}
		fmt.Printf("Auto register attempt %d failed: %v\n", i, err)
		time.Sleep(1 * time.Second)
	}

	fmt.Println("Auto register failed after retries")
}
