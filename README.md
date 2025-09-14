# 🔭 ContainerScope
**One Dashboard. All Your Containers. Everywhere.**

ContainerScope is a lightweight **multi-host Docker management dashboard** built with **Go + MySQL + React**.  
It allows you to connect to multiple remote Docker hosts, monitor containers, view logs, and track resource usage — all from a single UI.

---

## ✨ Features
- 🌍 Manage multiple Docker hosts
- 📊 Real-time container metrics (CPU, memory, network, disk)
- 📦 Start, stop, restart, remove containers
- 📝 View container logs
- 🗺️ Map view of servers by region
- 🔐 JWT authentication
- 🗄️ Persistent storage with MySQL

---

## 🛠️ Tech Stack
- **Backend:** Go (Golang) + Docker Remote API  
- **Database:** MySQL  
- **Frontend:** React + TailwindCSS + Chart.js + Leaflet/Mapbox  

---

## 🚀 Getting Started

### Prerequisites
- Docker installed on your remote hosts
- Go 1.20+  
- Node.js 18+  
- MySQL 8+

### Clone the repo
```bash
git clone https://github.com/your-username/containerscope.git
cd containerscope

