# DNS Application Lab – Go Implementation

This project implements a simplified DNS-based service discovery system using Go and Docker.

It consists of three services:

- **AS** – Authoritative Server (UDP DNS server, port 53333)
- **FS** – Fibonacci Server (HTTP server, port 9090)
- **US** – User Server (HTTP server, port 8080)

All services are containerized and orchestrated using Docker Compose.

---

## 1. Build and Run

From the root directory (`dns_app`), run:

```bash
docker compose up --build
```

This will:

- Build all three services
- Start them in a shared Docker network
- Automatically register the Fibonacci Server with the Authoritative Server

---

## 2. Test the System

Once the containers are running, open a browser and visit:

```
http://localhost:8080/fibonacci?hostname=fibonacci.com&fs_port=9090&number=7&as_ip=as&as_port=53333
```

Expected output:

```
13
```

---

## 3. How It Works

1. **FS (Fibonacci Server)** automatically registers itself with AS using UDP when it starts.
2. **US (User Server)**:
   - Queries AS via UDP to resolve the hostname.
   - Uses the returned address to contact FS via HTTP.
   - Returns the Fibonacci result to the client.

---

## 4. Ports Used

| Service | Protocol | Port  |
| ------- | -------- | ----- |
| AS      | UDP      | 53333 |
| FS      | HTTP     | 9090  |
| US      | HTTP     | 8080  |

---

## 5. Stop the System

To stop the containers:

```bash
docker compose down
```

---

## Notes

- No manual registration is required.
- All services communicate using Docker’s internal bridge network.
- Only the User Server (port 8080) needs to be accessed from the host machine.

```

```
