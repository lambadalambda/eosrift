package server

import (
	"net/http"
)

func serveLandingIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		return
	}

	_, _ = w.Write([]byte(landingIndexHTML))
}

func serveLandingStyle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		return
	}

	_, _ = w.Write([]byte(landingStyleCSS))
}

const landingIndexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Eosrift - Self-Hosted Tunnel Service</title>
    <link rel="stylesheet" href="style.css">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
</head>
<body>
    <div class="grid-bg"></div>

    <header class="header">
        <div class="container">
            <div class="logo">
                <span class="logo-icon">&#9670;</span>
                <span class="logo-text">Eosrift</span>
            </div>
            <nav class="nav">
                <a href="#features">Features</a>
                <a href="#quickstart">Quick Start</a>
                <a href="https://github.com/lambadalambda/eosrift" class="nav-github">GitHub</a>
            </nav>
        </div>
    </header>

    <main>
        <section class="hero">
            <div class="container">
                <div class="status-badge">
                    <span class="status-dot"></span>
                    <span>You are running Eosrift</span>
                </div>
                <h1 class="hero-title">
                    Your infrastructure.<br>
                    <span class="gradient-text">Your tunnels.</span>
                </h1>
                <p class="hero-subtitle">
                    Self-hosted tunnel service with ngrok-like simplicity.
                    Expose local services to the internet without vendor lock-in.
                </p>
                <div class="hero-actions">
                    <a href="#quickstart" class="btn btn-primary">Get Started</a>
                    <a href="https://github.com/lambadalambda/eosrift" class="btn btn-secondary">View Source</a>
                </div>

                <div class="hero-terminal">
                    <div class="terminal-header">
                        <div class="terminal-dots">
                            <span></span>
                            <span></span>
                            <span></span>
                        </div>
                        <span class="terminal-title">Terminal</span>
                    </div>
                    <div class="terminal-body">
                        <div class="terminal-line">
                            <span class="terminal-prompt">$</span>
                            <span class="terminal-cmd">eosrift http 3000</span>
                        </div>
                        <div class="terminal-output">
                            <span class="output-label">Eosrift</span>
                            <span class="output-version">v0.1.1</span>
                        </div>
                        <div class="terminal-output">
                            <span class="output-dim">Session Status</span>
                            <span class="output-success">online</span>
                        </div>
                        <div class="terminal-output">
                            <span class="output-dim">Forwarding</span>
                            <span class="output-url">https://abc123.tunnel.eosrift.com</span>
                            <span class="output-arrow">→</span>
                            <span>localhost:3000</span>
                        </div>
                        <div class="terminal-output">
                            <span class="output-dim">Inspector</span>
                            <span class="output-url">http://localhost:4040</span>
                        </div>
                    </div>
                </div>
            </div>
        </section>

        <section id="features" class="features">
            <div class="container">
                <h2 class="section-title">Built for developers who value control</h2>
                <div class="features-grid">
                    <div class="feature-card">
                        <div class="feature-icon">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>
                            </svg>
                        </div>
                        <h3>Self-Hosted</h3>
                        <p>Deploy on your own infrastructure. Your data never touches third-party servers.</p>
                    </div>
                    <div class="feature-card">
                        <div class="feature-icon">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <polyline points="4 17 10 11 4 5"/>
                                <line x1="12" y1="19" x2="20" y2="19"/>
                            </svg>
                        </div>
                        <h3>Familiar CLI</h3>
                        <p>ngrok-compatible commands and config. Switch without relearning anything.</p>
                    </div>
                    <div class="feature-card">
                        <div class="feature-icon">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <rect x="3" y="3" width="18" height="18" rx="2"/>
                                <path d="M3 9h18M9 21V9"/>
                            </svg>
                        </div>
                        <h3>HTTP &amp; TCP Tunnels</h3>
                        <p>Expose web apps, databases, SSH, or any TCP service through secure tunnels.</p>
                    </div>
                    <div class="feature-card">
                        <div class="feature-icon">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <circle cx="12" cy="12" r="10"/>
                                <path d="M12 6v6l4 2"/>
                            </svg>
                        </div>
                        <h3>Request Inspector</h3>
                        <p>Debug webhooks and API calls with the built-in request inspector at localhost:4040.</p>
                    </div>
                    <div class="feature-card">
                        <div class="feature-icon">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
                            </svg>
                        </div>
                        <h3>Docker-First</h3>
                        <p>Single docker-compose.yml deployment. Production-ready in minutes, not hours.</p>
                    </div>
                    <div class="feature-card">
                        <div class="feature-icon">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
                            </svg>
                        </div>
                        <h3>Automatic TLS</h3>
                        <p>Let's Encrypt certificates managed automatically. HTTPS by default, always.</p>
                    </div>
                </div>
            </div>
        </section>

        <section id="quickstart" class="quickstart">
            <div class="container">
                <h2 class="section-title">Up and running in minutes</h2>

                <div class="quickstart-group">
                    <h3 class="quickstart-group-title">On your server</h3>
                    <div class="quickstart-grid">
                        <div class="quickstart-step">
                            <div class="step-number">1</div>
                            <h3>Point your domains</h3>
                            <p class="step-desc">Point your base domain and wildcard tunnel subdomain to your server.</p>
                            <div class="code-block">
                                <code>eosrift.com → your.server.ip</code>
                                <code>*.tunnel.eosrift.com → your.server.ip</code>
                            </div>
                        </div>
                        <div class="quickstart-step">
                            <div class="step-number">2</div>
                            <h3>Configure and deploy</h3>
                            <p class="step-desc">Set your domain and run the stack.</p>
                            <div class="code-block">
                                <code>export EOSRIFT_BASE_DOMAIN=eosrift.com</code>
                                <code>export EOSRIFT_TUNNEL_DOMAIN=tunnel.eosrift.com</code>
                                <code>docker compose up -d --build</code>
                            </div>
                        </div>
                    </div>
                </div>

                <div class="quickstart-group">
                    <h3 class="quickstart-group-title">On your client</h3>
                    <div class="quickstart-grid">
                        <div class="quickstart-step">
                            <div class="step-number">1</div>
                            <h3>Configure your token</h3>
                            <div class="code-block">
                                <code>eosrift config add-authtoken &lt;your-token&gt;</code>
                            </div>
                        </div>
                        <div class="quickstart-step">
                            <div class="step-number">2</div>
                            <h3>Expose your service</h3>
                            <div class="code-block">
                                <code>eosrift http 8080</code>
                            </div>
                        </div>
                        <div class="quickstart-step">
                            <div class="step-number">3</div>
                            <h3>Share your URL</h3>
                            <div class="code-block">
                                <code>https://abc123.tunnel.eosrift.com</code>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </section>
    </main>

    <footer class="footer">
        <div class="container">
            <p class="footer-text">
                Eosrift is open source. Built with love for developers who value control.
            </p>
            <div class="footer-links">
                <a href="https://github.com/lambadalambda/eosrift">GitHub</a>
                <a href="#quickstart">Docs</a>
            </div>
        </div>
    </footer>
</body>
</html>
`

const landingStyleCSS = `/* ============================================
   Eosrift Landing Page Styles
   ============================================ */

:root {
    --bg-primary: #0a0a0f;
    --bg-secondary: #12121a;
    --bg-tertiary: #1a1a24;
    --surface: #16161f;
    --surface-hover: #1e1e2a;
    --border: #2a2a3a;
    --border-subtle: #1f1f2e;

    --text-primary: #f0f0f5;
    --text-secondary: #a0a0b0;
    --text-muted: #606070;

    --accent-primary: #6366f1;
    --accent-secondary: #8b5cf6;
    --accent-glow: rgba(99, 102, 241, 0.4);

    --gradient-start: #6366f1;
    --gradient-mid: #8b5cf6;
    --gradient-end: #a855f7;

    --success: #22c55e;
    --success-dim: #166534;

    --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
    --font-mono: 'JetBrains Mono', 'SF Mono', Consolas, monospace;

    --radius-sm: 6px;
    --radius-md: 10px;
    --radius-lg: 16px;
    --radius-xl: 24px;
}

*, *::before, *::after {
    box-sizing: border-box;
    margin: 0;
    padding: 0;
}

html {
    scroll-behavior: smooth;
}

body {
    font-family: var(--font-sans);
    background: var(--bg-primary);
    color: var(--text-primary);
    line-height: 1.6;
    overflow-x: hidden;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
}

/* Grid Background */
.grid-bg {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-image:
        linear-gradient(rgba(99, 102, 241, 0.03) 1px, transparent 1px),
        linear-gradient(90deg, rgba(99, 102, 241, 0.03) 1px, transparent 1px);
    background-size: 60px 60px;
    pointer-events: none;
    z-index: -1;
}

.container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 0 24px;
}

/* ============================================
   Header
   ============================================ */

.header {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    z-index: 100;
    padding: 20px 0;
    background: rgba(10, 10, 15, 0.8);
    backdrop-filter: blur(20px);
    border-bottom: 1px solid var(--border-subtle);
}

.header .container {
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.logo {
    display: flex;
    align-items: center;
    gap: 10px;
    text-decoration: none;
}

.logo-icon {
    font-size: 1.5rem;
    background: linear-gradient(135deg, var(--gradient-start), var(--gradient-end));
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
}

.logo-text {
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: -0.02em;
}

.nav {
    display: flex;
    align-items: center;
    gap: 32px;
}

.nav a {
    color: var(--text-secondary);
    text-decoration: none;
    font-size: 0.9rem;
    font-weight: 500;
    transition: color 0.2s ease;
}

.nav a:hover {
    color: var(--text-primary);
}

.nav-github {
    padding: 8px 16px;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    transition: all 0.2s ease;
}

.nav-github:hover {
    background: var(--surface-hover);
    border-color: var(--accent-primary);
}

/* ============================================
   Hero Section
   ============================================ */

.hero {
    min-height: 100vh;
    display: flex;
    align-items: center;
    padding: 120px 0 80px;
    position: relative;
}

.hero::before {
    content: '';
    position: absolute;
    top: 0;
    left: 50%;
    transform: translateX(-50%);
    width: 800px;
    height: 800px;
    background: radial-gradient(circle, var(--accent-glow) 0%, transparent 70%);
    opacity: 0.3;
    pointer-events: none;
}

.hero .container {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
}

.status-badge {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 100px;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-secondary);
    margin-bottom: 32px;
    animation: pulse-border 3s ease-in-out infinite;
}

@keyframes pulse-border {
    0%, 100% { border-color: var(--border); }
    50% { border-color: rgba(99, 102, 241, 0.6); }
}

.status-dot {
    width: 8px;
    height: 8px;
    background: var(--success);
    border-radius: 50%;
    box-shadow: 0 0 10px rgba(34, 197, 94, 0.6);
    animation: pulse-dot 2s ease-in-out infinite;
}

@keyframes pulse-dot {
    0%, 100% { transform: scale(1); opacity: 1; }
    50% { transform: scale(1.2); opacity: 0.8; }
}

.hero-title {
    font-size: clamp(3rem, 5vw, 4.5rem);
    font-weight: 700;
    line-height: 1.1;
    letter-spacing: -0.04em;
    margin-bottom: 24px;
}

.gradient-text {
    background: linear-gradient(135deg, var(--gradient-start), var(--gradient-mid), var(--gradient-end));
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
}

.hero-subtitle {
    font-size: 1.25rem;
    color: var(--text-secondary);
    max-width: 600px;
    margin-bottom: 40px;
}

.hero-actions {
    display: flex;
    gap: 16px;
    margin-bottom: 60px;
    flex-wrap: wrap;
    justify-content: center;
}

.btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 12px 24px;
    border-radius: var(--radius-md);
    font-weight: 600;
    text-decoration: none;
    transition: all 0.2s ease;
    font-size: 0.95rem;
}

.btn-primary {
    background: var(--accent-primary);
    color: white;
    box-shadow: 0 8px 25px rgba(99, 102, 241, 0.3);
}

.btn-primary:hover {
    background: #5558e8;
    transform: translateY(-2px);
    box-shadow: 0 12px 35px rgba(99, 102, 241, 0.4);
}

.btn-secondary {
    background: var(--surface);
    color: var(--text-primary);
    border: 1px solid var(--border);
}

.btn-secondary:hover {
    background: var(--surface-hover);
    border-color: var(--accent-primary);
    transform: translateY(-2px);
}

/* ============================================
   Terminal Preview
   ============================================ */

.hero-terminal {
    width: 100%;
    max-width: 700px;
    background: rgba(15, 23, 42, 0.4);
    border: 1px solid var(--border);
    border-radius: var(--radius-xl);
    overflow: hidden;
    box-shadow: 0 25px 50px rgba(0, 0, 0, 0.5);
    backdrop-filter: blur(20px);
}

.terminal-header {
    padding: 16px 20px;
    background: rgba(22, 22, 31, 0.8);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    gap: 12px;
}

.terminal-dots {
    display: flex;
    gap: 8px;
}

.terminal-dots span {
    width: 12px;
    height: 12px;
    border-radius: 50%;
}

.terminal-dots span:nth-child(1) { background: #ff5f57; }
.terminal-dots span:nth-child(2) { background: #ffbd2e; }
.terminal-dots span:nth-child(3) { background: #28ca42; }

.terminal-title {
    font-size: 0.875rem;
    color: var(--text-secondary);
    font-weight: 500;
}

.terminal-body {
    padding: 24px;
    font-family: var(--font-mono);
    text-align: left;
}

.terminal-line {
    display: flex;
    gap: 12px;
    margin-bottom: 20px;
    font-size: 1.1rem;
}

.terminal-prompt {
    color: var(--accent-primary);
    font-weight: 600;
}

.terminal-cmd {
    color: var(--text-primary);
}

.terminal-output {
    display: flex;
    gap: 16px;
    margin-bottom: 12px;
    font-size: 0.95rem;
    align-items: center;
}

.output-label {
    color: var(--accent-primary);
    font-weight: 600;
}

.output-version {
    color: var(--text-secondary);
    font-size: 0.9rem;
}

.output-dim {
    color: var(--text-muted);
    min-width: 120px;
}

.output-success {
    color: var(--success);
    font-weight: 500;
}

.output-url {
    color: var(--accent-primary);
    text-decoration: none;
    word-break: break-all;
}

.output-arrow {
    color: var(--text-muted);
}

/* ============================================
   Features Section
   ============================================ */

.features {
    padding: 100px 0;
    background: rgba(18, 18, 26, 0.3);
}

.section-title {
    font-size: 2.5rem;
    font-weight: 700;
    text-align: center;
    margin-bottom: 60px;
    letter-spacing: -0.03em;
}

.features-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 24px;
}

.feature-card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    padding: 32px;
    transition: all 0.3s ease;
    position: relative;
    overflow: hidden;
}

.feature-card::before {
    content: '';
    position: absolute;
    top: -2px;
    left: -2px;
    right: -2px;
    bottom: -2px;
    background: linear-gradient(135deg, rgba(99, 102, 241, 0.1), rgba(168, 85, 247, 0.1));
    opacity: 0;
    transition: opacity 0.3s ease;
    z-index: -1;
}

.feature-card:hover {
    transform: translateY(-4px);
    border-color: rgba(99, 102, 241, 0.4);
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
}

.feature-card:hover::before {
    opacity: 1;
}

.feature-icon {
    width: 48px;
    height: 48px;
    background: rgba(99, 102, 241, 0.1);
    border: 1px solid rgba(99, 102, 241, 0.2);
    border-radius: var(--radius-md);
    display: flex;
    align-items: center;
    justify-content: center;
    margin: 0 auto 20px;
}

.feature-icon svg {
    width: 24px;
    height: 24px;
    color: var(--accent-primary);
}

.feature-card h3 {
    font-size: 1.25rem;
    font-weight: 600;
    margin-bottom: 12px;
    text-align: center;
}

.feature-card p {
    color: var(--text-secondary);
    text-align: center;
    font-size: 0.95rem;
    line-height: 1.6;
}

/* ============================================
   Quickstart Section
   ============================================ */

.quickstart {
    padding: 100px 0;
}

.quickstart-group {
    margin-bottom: 80px;
}

.quickstart-group-title {
    font-size: 1.75rem;
    font-weight: 600;
    margin-bottom: 32px;
    text-align: center;
    color: var(--text-primary);
}

.quickstart-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 24px;
}

.quickstart-step {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    padding: 32px;
    position: relative;
}

.step-number {
    width: 32px;
    height: 32px;
    background: var(--accent-primary);
    color: white;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 600;
    margin: 0 auto 20px;
}

.quickstart-step h3 {
    font-size: 1.2rem;
    margin-bottom: 12px;
    text-align: center;
}

.step-desc {
    color: var(--text-secondary);
    text-align: center;
    margin-bottom: 20px;
    font-size: 0.95rem;
}

.code-block {
    background: var(--bg-primary);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    padding: 16px;
    font-family: var(--font-mono);
    font-size: 0.85rem;
}

.code-block code {
    display: block;
    color: var(--text-primary);
    margin-bottom: 8px;
}

.code-block code:last-child {
    margin-bottom: 0;
}

/* ============================================
   Footer
   ============================================ */

.footer {
    padding: 60px 0;
    border-top: 1px solid var(--border-subtle);
    background: rgba(18, 18, 26, 0.3);
}

.footer .container {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 20px;
}

.footer-text {
    color: var(--text-secondary);
    font-size: 0.9rem;
}

.footer-links {
    display: flex;
    gap: 24px;
}

.footer-links a {
    color: var(--text-secondary);
    text-decoration: none;
    font-size: 0.9rem;
    transition: color 0.2s ease;
}

.footer-links a:hover {
    color: var(--accent-primary);
}

/* ============================================
   Responsive Design
   ============================================ */

@media (max-width: 768px) {
    .nav {
        display: none;
    }

    .hero-title {
        font-size: 3rem;
    }

    .section-title {
        font-size: 2rem;
    }

    .hero-terminal {
        margin: 0 -12px;
    }

    .terminal-body {
        padding: 16px;
    }

    .terminal-output {
        flex-wrap: wrap;
        gap: 8px;
    }

    .output-dim {
        min-width: auto;
    }

    .footer .container {
        flex-direction: column;
        text-align: center;
    }
}

@media (max-width: 480px) {
    .container {
        padding: 0 16px;
    }

    .hero-actions {
        flex-direction: column;
        width: 100%;
    }

    .btn {
        width: 100%;
    }
}
`
