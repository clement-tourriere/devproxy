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
	Project  string               `json:"project"`
	Service  string               `json:"service"`
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
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; 
            margin: 0; 
            background: #f5f6fa;
        }
        .main-container { 
            display: flex; 
            min-height: 100vh;
        }
        .sidebar {
            width: 280px;
            background: white;
            border-right: 1px solid #e0e6ed;
            position: fixed;
            height: 100vh;
            overflow-y: auto;
            z-index: 100;
        }
        .sidebar-content {
            padding: 20px;
        }
        .sidebar h2 {
            font-size: 1.1em;
            color: #2c3e50;
            margin: 0 0 15px 0;
            border-bottom: 1px solid #e0e6ed;
            padding-bottom: 10px;
        }
        .nav-item {
            padding: 8px 12px;
            margin: 2px 0;
            border-radius: 6px;
            cursor: pointer;
            font-size: 0.9em;
            transition: background 0.2s;
        }
        .nav-item:hover {
            background: #f8f9fa;
        }
        .nav-item.project {
            font-weight: 600;
            color: #2c3e50;
        }
        .nav-item.standalone {
            color: #6c757d;
        }
        .nav-count {
            float: right;
            background: #e9ecef;
            color: #6c757d;
            padding: 2px 8px;
            border-radius: 12px;
            font-size: 0.8em;
        }
        .content {
            margin-left: 280px;
            padding: 30px 40px;
            width: calc(100% - 280px);
        }
        .header {
            margin-bottom: 30px;
        }
        h1 { 
            color: #2c3e50; 
            margin: 0 0 10px 0; 
            font-size: 1.8em;
        }
        .search-container {
            display: flex;
            gap: 10px;
            margin-bottom: 20px;
            align-items: center;
        }
        .search-input {
            flex: 1;
            padding: 10px 15px;
            border: 1px solid #d1d9e0;
            border-radius: 8px;
            font-size: 0.9em;
        }
        .filter-buttons {
            display: flex;
            gap: 5px;
        }
        .filter-btn {
            padding: 8px 12px;
            border: 1px solid #d1d9e0;
            background: white;
            border-radius: 6px;
            cursor: pointer;
            font-size: 0.85em;
            transition: all 0.2s;
        }
        .filter-btn.active {
            background: #007bff;
            color: white;
            border-color: #007bff;
        }
        .refresh-info { 
            color: #6c757d; 
            font-size: 0.9em; 
            margin-bottom: 25px; 
        }
        .protocol-status { 
            margin-bottom: 25px; 
            padding: 15px; 
            border-radius: 8px; 
        }
        .protocol-https { 
            background: #d4edda; 
            border: 1px solid #c3e6cb; 
            color: #155724; 
        }
        .protocol-http { 
            background: #f8d7da; 
            border: 1px solid #f5c6cb; 
            color: #721c24; 
        }
        .protocol-status a { 
            color: inherit; 
            font-weight: bold; 
            text-decoration: underline; 
            cursor: pointer; 
        }
        .install-instructions { 
            margin-top: 10px; 
            padding: 15px; 
            background: #f8f9fa; 
            border-radius: 4px; 
            border-left: 3px solid #007bff; 
            display: none; 
        }
        .install-instructions.show { 
            display: block; 
        }
        .install-instructions h4 { 
            margin: 0 0 10px 0; 
            color: #333; 
        }
        .install-instructions ol { 
            margin: 0; 
            padding-left: 20px; 
        }
        .install-instructions li { 
            margin: 5px 0; 
        }
        .project-group { 
            margin-bottom: 25px; 
            background: white;
            border-radius: 12px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.08);
            overflow: hidden;
        }
        .project-header { 
            padding: 15px 20px;
            background: #f8f9fa;
            border-bottom: 1px solid #e9ecef;
            cursor: pointer;
            display: flex;
            justify-content: space-between;
            align-items: center;
            transition: background 0.2s;
        }
        .project-header:hover {
            background: #e9ecef;
        }
        .project-title {
            font-size: 1.1em;
            font-weight: 600;
            color: #2c3e50;
        }
        .project-controls {
            display: flex;
            gap: 10px;
            align-items: center;
        }
        .collapse-icon {
            font-size: 0.8em;
            color: #6c757d;
            transition: transform 0.2s;
        }
        .project-group.collapsed .collapse-icon {
            transform: rotate(-90deg);
        }
        .container-count {
            background: #e3f2fd;
            color: #1976d2;
            padding: 4px 10px;
            border-radius: 12px;
            font-size: 0.8em;
            font-weight: 500;
        }
        .containers-table {
            display: none;
        }
        .project-group:not(.collapsed) .containers-table {
            display: block;
        }
        .container-row {
            display: flex;
            align-items: center;
            padding: 12px 20px;
            border-bottom: 1px solid #f1f3f4;
            transition: background 0.2s;
        }
        .container-row:hover {
            background: #f8f9fa;
        }
        .container-row:last-child {
            border-bottom: none;
        }
        .status-indicator {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            margin-right: 12px;
        }
        .status-running { background: #28a745; }
        .status-starting { background: #ffc107; }
        .status-stopped { background: #dc3545; }
        .container-info {
            flex: 1;
            min-width: 0;
        }
        .container-name {
            font-weight: 600;
            color: #2c3e50;
            margin-bottom: 2px;
        }
        .container-meta {
            font-size: 0.85em;
            color: #6c757d;
        }
        .container-actions {
            display: flex;
            gap: 8px;
            align-items: center;
        }
        .link-button {
            background: #007bff;
            color: white;
            padding: 6px 12px;
            border-radius: 6px;
            text-decoration: none;
            font-size: 0.85em;
            font-weight: 500;
            transition: background 0.2s;
        }
        .link-button:hover {
            background: #0056b3;
            text-decoration: none;
        }
        .copy-button {
            background: #6c757d;
            color: white;
            border: none;
            padding: 6px 10px;
            border-radius: 6px;
            cursor: pointer;
            font-size: 0.8em;
            transition: background 0.2s;
        }
        .copy-button:hover {
            background: #545b62;
        }
        .no-containers { 
            text-align: center; 
            color: #6c757d; 
            padding: 40px; 
            background: white;
            border-radius: 12px;
        }
        .expand-all-btn {
            position: fixed;
            bottom: 30px;
            right: 30px;
            background: #007bff;
            color: white;
            border: none;
            padding: 12px 20px;
            border-radius: 25px;
            cursor: pointer;
            font-weight: 500;
            box-shadow: 0 4px 12px rgba(0,123,255,0.3);
            transition: all 0.2s;
        }
        .expand-all-btn:hover {
            background: #0056b3;
            transform: translateY(-2px);
        }
    </style>
    <script>
        let instructionsVisible = false;
        let currentProtocol = window.location.protocol; // 'http:' or 'https:'
        let allContainers = [];
        let filteredContainers = [];
        let currentFilter = 'all';
        let searchQuery = '';
        
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
                    allContainers = containers;
                    applyFilters();
                    renderContainers();
                    renderSidebar();
                })
                .catch(err => console.error('Failed to load containers:', err));
        }
        
        function applyFilters() {
            filteredContainers = allContainers.filter(c => {
                // Status filter
                if (currentFilter === 'running' && c.status !== 'running') return false;
                if (currentFilter === 'stopped' && c.status === 'running') return false;
                
                // Search filter
                if (searchQuery) {
                    const query = searchQuery.toLowerCase();
                    const searchableText = [
                        c.name || '',
                        c.service || '',
                        c.project || '',
                        c.image || '',
                        ...(c.targets?.map(t => t.Domain) || [])
                    ].join(' ').toLowerCase();
                    
                    if (!searchableText.includes(query)) return false;
                }
                
                return true;
            });
        }
        
        function renderSidebar() {
            const sidebar = document.getElementById('sidebar-content');
            
            // Group containers by project for sidebar
            const grouped = {};
            const standalone = [];
            
            filteredContainers.forEach(c => {
                if (c.project) {
                    if (!grouped[c.project]) {
                        grouped[c.project] = [];
                    }
                    grouped[c.project].push(c);
                } else {
                    standalone.push(c);
                }
            });
            
            let html = '<h2>Navigation</h2>';
            
            // Projects
            Object.keys(grouped).sort().forEach(projectName => {
                html += '<div class="nav-item project" onclick="scrollToProject(\'' + projectName + '\')">';
                html += '🐳 ' + projectName;
                html += '<span class="nav-count">' + grouped[projectName].length + '</span>';
                html += '</div>';
            });
            
            // Standalone containers
            if (standalone.length > 0) {
                html += '<div class="nav-item project" onclick="scrollToProject(\'standalone\')">';
                html += '📦 Standalone';
                html += '<span class="nav-count">' + standalone.length + '</span>';
                html += '</div>';
            }
            
            sidebar.innerHTML = html;
        }

        function renderContainers() {
            const container = document.getElementById('containers');
            if (filteredContainers.length === 0) {
                container.innerHTML = '<div class="no-containers">No containers found matching your filters</div>';
                return;
            }
            
            // Group containers by project
            const grouped = {};
            const standalone = [];
            
            filteredContainers.forEach(c => {
                if (c.project) {
                    if (!grouped[c.project]) {
                        grouped[c.project] = [];
                    }
                    grouped[c.project].push(c);
                } else {
                    standalone.push(c);
                }
            });
            
            let html = '';
            
            // Render compose projects
            Object.keys(grouped).sort().forEach(projectName => {
                html += renderProjectGroup(projectName, grouped[projectName], '🐳', 'Compose Project');
            });
            
            // Render standalone containers
            if (standalone.length > 0) {
                html += renderProjectGroup('standalone', standalone, '📦', 'Standalone Containers');
            }
            
            container.innerHTML = html;
        }
        
        function renderProjectGroup(projectName, containers, icon, subtitle) {
            const projectId = 'project-' + projectName.replace(/[^a-zA-Z0-9]/g, '');
            const isCollapsed = localStorage.getItem(projectId + '-collapsed') === 'true';
            
            let html = '<div class="project-group' + (isCollapsed ? ' collapsed' : '') + '" id="' + projectId + '">';
            html += '<div class="project-header" onclick="toggleProject(\'' + projectId + '\')">';
            html += '<div class="project-title">' + icon + ' ' + projectName;
            if (subtitle !== projectName) {
                html += ' <span style="font-weight: normal; color: #6c757d;">(' + subtitle + ')</span>';
            }
            html += '</div>';
            html += '<div class="project-controls">';
            html += '<span class="container-count">' + containers.length + ' service' + (containers.length !== 1 ? 's' : '') + '</span>';
            html += '<span class="collapse-icon">▼</span>';
            html += '</div>';
            html += '</div>';
            
            html += '<div class="containers-table">';
            containers.forEach(c => {
                html += renderContainerRow(c);
            });
            html += '</div>';
            
            html += '</div>';
            return html;
        }
        
        function renderContainerRow(c) {
            const protocol = currentProtocol.replace(':', '');
            const displayName = c.service || c.name || 'Unknown';
            const primaryDomain = c.targets && c.targets.length > 0 ? c.targets[0].Domain : '';
            const statusClass = 'status-' + (c.status === 'running' ? 'running' : c.status === 'starting' ? 'starting' : 'stopped');
            
            let html = '<div class="container-row">';
            html += '<div class="status-indicator ' + statusClass + '"></div>';
            html += '<div class="container-info">';
            html += '<div class="container-name">' + displayName + '</div>';
            html += '<div class="container-meta">' + (c.image || 'Unknown image');
            if (c.service && c.name !== c.service) {
                html += ' • ' + c.name;
            }
            html += '</div>';
            html += '</div>';
            
            if (primaryDomain) {
                html += '<div class="container-actions">';
                html += '<a href="' + protocol + '://' + primaryDomain + '" target="_blank" class="link-button">Open</a>';
                html += '<button onclick="copyToClipboard(event, \'' + protocol + '://' + primaryDomain + '\')" class="copy-button">Copy</button>';
                html += '</div>';
            }
            
            html += '</div>';
            return html;
        }
        
        function toggleProject(projectId) {
            const projectGroup = document.getElementById(projectId);
            const isCollapsed = projectGroup.classList.contains('collapsed');
            
            if (isCollapsed) {
                projectGroup.classList.remove('collapsed');
                localStorage.setItem(projectId + '-collapsed', 'false');
            } else {
                projectGroup.classList.add('collapsed');
                localStorage.setItem(projectId + '-collapsed', 'true');
            }
        }
        
        function scrollToProject(projectName) {
            const projectId = 'project-' + projectName.replace(/[^a-zA-Z0-9]/g, '');
            const element = document.getElementById(projectId);
            if (element) {
                element.scrollIntoView({ behavior: 'smooth', block: 'start' });
                // Expand if collapsed
                if (element.classList.contains('collapsed')) {
                    toggleProject(projectId);
                }
            }
        }
        
        function setFilter(filter) {
            currentFilter = filter;
            // Update button states
            document.querySelectorAll('.filter-btn').forEach(btn => {
                btn.classList.remove('active');
            });
            document.querySelector('[onclick="setFilter(\'' + filter + '\')"]').classList.add('active');
            
            applyFilters();
            renderContainers();
            renderSidebar();
        }
        
        function handleSearch(query) {
            searchQuery = query;
            applyFilters();
            renderContainers();
            renderSidebar();
        }
        
        function expandAll() {
            const allProjects = document.querySelectorAll('.project-group');
            const hasCollapsed = Array.from(allProjects).some(p => p.classList.contains('collapsed'));
            
            allProjects.forEach(project => {
                if (hasCollapsed) {
                    project.classList.remove('collapsed');
                    localStorage.setItem(project.id + '-collapsed', 'false');
                } else {
                    project.classList.add('collapsed');
                    localStorage.setItem(project.id + '-collapsed', 'true');
                }
            });
            
            // Update button text
            const btn = document.querySelector('.expand-all-btn');
            btn.textContent = hasCollapsed ? 'Collapse All' : 'Expand All';
        }
        
        function copyToClipboard(event, text) {
            navigator.clipboard.writeText(text).then(() => {
                // Show a brief success indicator
                const button = event.target;
                const originalText = button.textContent;
                const originalBackground = button.style.background;
                
                // Update button appearance
                button.textContent = 'Copied!';
                button.style.background = '#28a745';
                button.style.transform = 'scale(0.95)';
                button.style.transition = 'all 0.2s ease';
                
                // Reset after delay
                setTimeout(() => {
                    button.textContent = originalText;
                    button.style.background = originalBackground;
                    button.style.transform = '';
                }, 1500);
            }).catch(err => {
                // Fallback for older browsers or permission issues
                console.error('Failed to copy to clipboard:', err);
                const button = event.target;
                const originalText = button.textContent;
                
                button.textContent = 'Failed!';
                button.style.background = '#dc3545';
                setTimeout(() => {
                    button.textContent = originalText;
                    button.style.background = '';
                }, 1500);
            });
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
    <div class="main-container">
        <div class="sidebar">
            <div class="sidebar-content" id="sidebar-content">
                <h2>Navigation</h2>
                <div class="nav-item">Loading...</div>
            </div>
        </div>
        
        <div class="content">
            <div class="header">
                <h1>DevProxy Dashboard</h1>
                <div id="protocol-status" class="protocol-status"></div>
                
                <div class="search-container">
                    <input type="text" class="search-input" placeholder="Search containers, projects, domains..." 
                           oninput="handleSearch(this.value)">
                    <div class="filter-buttons">
                        <button class="filter-btn active" onclick="setFilter('all')">All</button>
                        <button class="filter-btn" onclick="setFilter('running')">Running</button>
                        <button class="filter-btn" onclick="setFilter('stopped')">Stopped</button>
                    </div>
                </div>
                
                <div class="refresh-info">Auto-refresh every 30 seconds</div>
            </div>
            
            <div id="containers">Loading...</div>
        </div>
        
        <button class="expand-all-btn" onclick="expandAll()">Expand All</button>
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

		// Extract compose project and service information
		project := ""
		service := ""
		if containerInfo.Config != nil && containerInfo.Config.Labels != nil {
			if projectName, exists := containerInfo.Config.Labels["com.docker.compose.project"]; exists {
				project = projectName
			}
			if serviceName, exists := containerInfo.Config.Labels["com.docker.compose.service"]; exists {
				service = serviceName
			}
		}

		container := ContainerInfo{
			ID:       containerInfo.ID[:12], // Shortened ID
			Name:     containerName,
			Image:    image,
			Status:   status,
			Targets:  targets,
			Protocol: "", // Will be determined by frontend based on current location
			Project:  project,
			Service:  service,
		}
		containers = append(containers, container)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(containers)
}
