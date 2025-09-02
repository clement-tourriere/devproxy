package dashboard

import (
	"context"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"devproxy/internal/docker"
	"devproxy/internal/proxy"
)

type Server struct {
	manager *proxy.Manager
	logger  *slog.Logger
}

type ContainerInfo struct {
	ID       string               `json:"id"`
	Name     string               `json:"name"`
	Image    string               `json:"image"`
	Status   string               `json:"status"`
	Targets  []docker.ProxyTarget `json:"targets"`
	Protocol string               `json:"protocol"`
}

func NewServer(manager *proxy.Manager, logger *slog.Logger) *Server {
	return &Server{
		manager: manager,
		logger:  logger,
	}
}

func (s *Server) Start(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleDashboard)
	mux.HandleFunc("/api/containers", s.handleAPIContainers)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	s.logger.Info("Starting dashboard server", "addr", addr)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>DevProxy Dashboard</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 40px; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 30px; }
        .container-card { background: #f8f9fa; border-radius: 8px; padding: 20px; margin-bottom: 20px; border-left: 4px solid #007bff; }
        .container-name { font-size: 1.2em; font-weight: bold; margin-bottom: 10px; }
        .container-meta { color: #666; font-size: 0.9em; margin-bottom: 15px; }
        .targets { margin-top: 15px; }
        .target { background: white; padding: 10px 15px; margin: 5px 0; border-radius: 4px; border: 1px solid #e0e0e0; }
        .target a { color: #007bff; text-decoration: none; font-weight: 500; }
        .target a:hover { text-decoration: underline; }
        .no-containers { text-align: center; color: #666; padding: 40px; }
        .refresh-info { color: #666; font-size: 0.9em; margin-bottom: 20px; }
        .protocol-status { margin-bottom: 20px; padding: 15px; border-radius: 8px; }
        .protocol-https { background: #d4edda; border: 1px solid #c3e6cb; color: #155724; }
        .protocol-http { background: #f8d7da; border: 1px solid #f5c6cb; color: #721c24; }
        .protocol-status a { color: inherit; font-weight: bold; text-decoration: underline; cursor: pointer; }
        .install-instructions { margin-top: 10px; padding: 15px; background: #f8f9fa; border-radius: 4px; border-left: 3px solid #007bff; display: none; }
        .install-instructions.show { display: block; }
        .install-instructions h4 { margin: 0 0 10px 0; color: #333; }
        .install-instructions ol { margin: 0; padding-left: 20px; }
        .install-instructions li { margin: 5px 0; }
    </style>
    <script>
        let instructionsVisible = false;
        let currentProtocol = window.location.protocol; // 'http:' or 'https:'
        
        function loadProtocolStatus() {
            const statusDiv = document.getElementById('protocol-status');
            if (currentProtocol === 'https:') {
                statusDiv.className = 'protocol-status protocol-https';
                statusDiv.innerHTML = '✅ Using secure HTTPS connections - certificates are trusted';
            } else {
                statusDiv.className = 'protocol-status protocol-http';
                statusDiv.innerHTML = '⚠️ Using HTTP connections. <a href="#" onclick="toggleInstallInstructions(); return false;">Enable HTTPS</a>' +
                    '<div id="install-instructions" class="install-instructions' + (instructionsVisible ? ' show' : '') + '">' +
                        '<h4>To enable trusted HTTPS connections:</h4>' +
                        '<ol>' +
                            '<li>Run: <code>./trust-cert.sh</code></li>' +
                            '<li>Restart your browser</li>' +
                            '<li>Access: <a href="https://devproxy-dashboard.localhost">https://devproxy-dashboard.localhost</a></li>' +
                        '</ol>' +
                        '<p><strong>Note:</strong> The script works on macOS, Linux, and Windows.</p>' +
                    '</div>';
            }
        }

        function loadContainers() {
            fetch('/api/containers')
                .then(response => response.json())
                .then(containers => {
                    const container = document.getElementById('containers');
                    if (containers.length === 0) {
                        container.innerHTML = '<div class="no-containers">No active containers found</div>';
                        return;
                    }
                    
                    container.innerHTML = containers.map(c => {
                        const protocol = currentProtocol.replace(':', ''); // Use current browser protocol
                        const targets = c.targets.map(t => 
                            '<div class="target"><a href="' + protocol + '://' + t.Domain + '" target="_blank">' + t.Domain + '</a> → ' + t.ContainerIP + ':' + t.Port + '</div>'
                        ).join('');
                        
                        return '<div class="container-card">' +
                            '<div class="container-name">' + (c.name || 'Unknown') + '</div>' +
                            '<div class="container-meta">Image: ' + (c.image || 'Unknown') + ' | Status: ' + (c.status || 'Unknown') + ' | Protocol: ' + protocol.toUpperCase() + '</div>' +
                            '<div class="targets">' + targets + '</div>' +
                        '</div>';
                    }).join('');
                })
                .catch(err => console.error('Failed to load containers:', err));
        }

        function toggleInstallInstructions() {
            instructionsVisible = !instructionsVisible;
            const instructions = document.getElementById('install-instructions');
            if (instructions) {
                if (instructionsVisible) {
                    instructions.classList.add('show');
                } else {
                    instructions.classList.remove('show');
                }
            }
        }
        
        // Load initially and refresh every 30 seconds
        document.addEventListener('DOMContentLoaded', function() {
            loadProtocolStatus();
            loadContainers();
            setInterval(function() {
                loadProtocolStatus();
                loadContainers();
            }, 30000);
        });
    </script>
</head>
<body>
    <div class="container">
        <h1>DevProxy Dashboard</h1>
        <div id="protocol-status" class="protocol-status"></div>
        <div class="refresh-info">Auto-refresh every 30 seconds</div>
        <div id="containers">Loading...</div>
    </div>
</body>
</html>`

	t, err := template.New("dashboard").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, nil)
}

func (s *Server) handleAPIContainers(w http.ResponseWriter, r *http.Request) {

	// Get containers directly from Docker since the dashboard manager isn't started
	dockerContainers, err := s.manager.GetRunningContainers(context.Background())
	if err != nil {
		s.logger.Error("Failed to get running containers", "error", err)
		http.Error(w, "Failed to get containers", http.StatusInternalServerError)
		return
	}

	var containers []ContainerInfo
	for _, dockerContainer := range dockerContainers {
		// Inspect each container to get full details
		containerInfo, err := s.manager.InspectContainer(context.Background(), dockerContainer.ID)
		if err != nil {
			s.logger.Warn("Failed to inspect container", "container_id", dockerContainer.ID, "error", err)
			continue
		}

		// Use discovery logic to extract proxy targets
		targets := s.manager.GetDiscovery().ExtractProxyTargets(containerInfo)
		if len(targets) == 0 {
			continue
		}

		// Extract container name from the container info
		containerName := strings.TrimPrefix(containerInfo.Name, "/")

		// Handle potential nil pointers
		image := "Unknown"
		if containerInfo.Config != nil && containerInfo.Config.Image != "" {
			image = containerInfo.Config.Image
		}

		status := "Unknown"
		if containerInfo.State != nil && containerInfo.State.Status != "" {
			status = containerInfo.State.Status
		}

		container := ContainerInfo{
			ID:       containerInfo.ID[:12], // Shortened ID
			Name:     containerName,
			Image:    image,
			Status:   status,
			Targets:  targets,
			Protocol: "", // Will be determined by frontend based on current location
		}
		containers = append(containers, container)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(containers)
}
