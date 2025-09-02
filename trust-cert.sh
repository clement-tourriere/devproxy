#!/bin/bash
set -e

# DevProxy Certificate Trust Installer
# Installs Caddy's root certificate in the system trust store
# Works on macOS, Linux, and Windows

echo "🔐 Installing DevProxy HTTPS certificate..."

# Check if DevProxy is running
if ! docker ps | grep -q devproxy-caddy; then
    echo "❌ Error: DevProxy containers not running. Start with 'docker compose up -d' first."
    exit 1
fi

# Export certificate from container
echo "📜 Exporting certificate from Caddy..."
if ! docker exec devproxy-caddy cat /data/caddy/pki/authorities/local/root.crt > /tmp/caddy-root.crt 2>/dev/null; then
    echo "❌ Error: Could not export certificate from Caddy container."
    exit 1
fi

# Detect operating system and install certificate
case "$(uname -s)" in
    Darwin*)
        echo "🍎 Detected macOS - installing in System Keychain..."
        if sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain /tmp/caddy-root.crt 2>/dev/null; then
            echo "✅ Certificate installed successfully in macOS Keychain!"
        else
            echo "❌ Failed to install certificate. Make sure you have admin privileges."
            rm -f /tmp/caddy-root.crt
            exit 1
        fi
        ;;
    
    Linux*)
        echo "🐧 Detected Linux - installing in system trust store..."
        
        # Check for different Linux distributions
        if command -v update-ca-certificates >/dev/null 2>&1; then
            # Debian/Ubuntu family
            echo "   Using update-ca-certificates (Debian/Ubuntu)..."
            sudo cp /tmp/caddy-root.crt /usr/local/share/ca-certificates/caddy-devproxy.crt
            sudo update-ca-certificates
            echo "✅ Certificate installed successfully in Linux trust store!"
            
        elif command -v update-ca-trust >/dev/null 2>&1; then
            # RedHat/Fedora/CentOS family
            echo "   Using update-ca-trust (RHEL/Fedora/CentOS)..."
            sudo cp /tmp/caddy-root.crt /etc/pki/ca-trust/source/anchors/caddy-devproxy.crt
            sudo update-ca-trust
            echo "✅ Certificate installed successfully in Linux trust store!"
            
        elif command -v trust >/dev/null 2>&1; then
            # p11-kit trust (modern Linux)
            echo "   Using p11-kit trust..."
            sudo trust anchor --store /tmp/caddy-root.crt
            echo "✅ Certificate installed successfully in Linux trust store!"
            
        else
            echo "❌ Unsupported Linux distribution or missing certificate tools."
            echo "   Please install ca-certificates package or consult your distribution's documentation."
            rm -f /tmp/caddy-root.crt
            exit 1
        fi
        ;;
    
    MINGW*|CYGWIN*|MSYS*)
        echo "🪟 Detected Windows environment..."
        if command -v certutil.exe >/dev/null 2>&1; then
            certutil.exe -addstore -f "ROOT" /tmp/caddy-root.crt
            echo "✅ Certificate installed successfully in Windows trust store!"
        else
            echo "❌ certutil not found. Please run this from an Administrator Command Prompt."
            rm -f /tmp/caddy-root.crt
            exit 1
        fi
        ;;
    
    *)
        echo "❌ Unsupported operating system: $(uname -s)"
        echo "   Please manually install the certificate from /tmp/caddy-root.crt"
        exit 1
        ;;
esac

# Clean up
rm -f /tmp/caddy-root.crt

echo ""
echo "🎉 Success! DevProxy HTTPS certificates are now trusted."
echo "   You can now visit https://container-name.localhost without warnings!"
echo ""
echo "💡 Note: You may need to restart your browser for the changes to take effect."