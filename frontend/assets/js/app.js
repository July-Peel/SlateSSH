function shadowApp() {
  const cachedAuth = localStorage.getItem('slate_auth') === 'true';
  let cachedConnections = [];
  try {
    cachedConnections = JSON.parse(localStorage.getItem('slate_connections_cache') || '[]');
  } catch (_) {}

  return {
    booting: !cachedAuth,
    needsSetup: false,
    authenticated: cachedAuth,
    currentUser: null,
    error: '',
    connections: cachedConnections,
    sessions: [],
    spinnerFrames: ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'],
    spinnerIndex: 0,
    currentSpinner: '⠋',
    spinnerTimer: null,
    terminals: {},
    rdpClients: {},
    rdpKeyboards: {},
    rdpViewportWidth: 1280,
    rdpViewportHeight: 720,
    searchAddons: {},
    fitAddons: {},
    socket: null,
    socketOpenPromise: null,
    activeSessionId: null,
    activePath: '.',
    files: [],
    statuses: {},
    theme: localStorage.getItem('slatessh_color_mode') || localStorage.getItem('slatessh-theme') || 'dark',
    colorMode: localStorage.getItem('slatessh_color_mode') || localStorage.getItem('slatessh-theme') || 'dark',
    palette: localStorage.getItem('slatessh_palette') || 'blue',
    showThemePicker: false,
    showPassword: false,
    availablePalettes: [
      { id: 'blue', name: 'Google Blue', color: '#0b57d0' },
      { id: 'green', name: 'Emerald Green', color: '#1e8e3e' },
      { id: 'amber', name: 'Sunset Amber', color: '#d97706' },
      { id: 'red', name: 'Expressive Red', color: '#d93025' },
      { id: 'purple', name: 'Iris Purple', color: '#6750a4' },
      { id: 'teal', name: 'Ocean Teal', color: '#006a6a' },
      { id: 'pink', name: 'Rose Pink', color: '#984061' }
    ],
    rightTab: 'files',
    showSidebar: true,
    showStatusWidget: true,
    showSftpWidget: true,
    pathInput: '',
    showPathDropdown: false,
    activeSuggestionIndex: -1,
    pathHistory: JSON.parse(localStorage.getItem('slatessh-path-history') || '[]'),
    showEditorPanel: false,
    showConnectionForm: false,
    showConnectionManager: false,
    connectionFormSource: null, // 'launcher' | 'sidebar' | 'directory'
    isFullscreen: false,
    connectionQuery: '',
    terminalSearch: '',
    testMessage: '',
    testMessageType: 'info',
    testResultModal: { visible: false, title: '', message: '', type: 'info' },
    editorTabs: [],
    activeEditorTabId: null,
    contextMenu: { visible: false, x: 0, y: 0, entry: null },
    clipboard: { mode: null, entries: [] },
    uploadProgress: 0,
    uploading: false,
    pendingPasteTarget: null,
    terminalPasteBox: { visible: false, text: '' },
    terminalCopyViewer: { visible: false, text: '' },
    editorSaveFeedback: { state: 'idle', path: '', timer: null },
    loginStep: 'username', // 'username' | 'password'
    editorWindow: {
      x: 420,
      y: 90,
      width: 760,
      height: 520,
      dragging: false,
      resizing: false,
      resizeEdge: '',
      offsetX: 0,
      offsetY: 0,
      startX: 0,
      startY: 0,
      startWidth: 0,
      startHeight: 0,
      startLeft: 0,
      startTop: 0,
      maximized: false,
      minimized: false,
      restore: null
    },
    monacoInstance: null,
    monacoReady: false,
    monacoInitTimer: null,
    isMonacoUpdating: false,
    setupForm: { username: '', password: '', confirmPassword: '' },
    loginForm: { username: '', password: '', rememberMe: true },
    connectionForm: { id: null, name: '', type: 'SSH', host: '', port: 22, username: 'root', auth_method: 'password', password: '', private_key: '', passphrase: '', notes: '' },
    isMobile: window.matchMedia('(max-width: 1023px), (pointer: coarse) and (max-height: 500px)').matches,
    isOnline: navigator.onLine,
    isStandalone: window.matchMedia('(display-mode: standalone)').matches || navigator.standalone === true,
    secureContext: window.isSecureContext,
    showInstallHelp: false,
    updateAvailable: false,
    authSubmitting: false,
    showMobileTools: false,
    terminalObservers: {},
    isKeyboardOpen: false,
    showMobileSftp: false,
    showMobileStatus: false,
    showMobileMenu: false,
    ctrlKeyActive: false,

    focusLoginInput(force = false) {
      if ((this.isMobile && !force) || this.authenticated || this.booting || this.needsSetup) return;
      this.$nextTick(() => {
        const targetId = this.loginStep === 'password' ? 'term-login-password' : 'term-login-username';
        document.getElementById(targetId)?.focus({ preventScroll: true });
      });
    },

    handleUsernameSubmit() {
      const username = (this.loginForm.username || '').trim();
      if (!username) {
        this.focusLoginInput(true);
        return;
      }
      this.loginStep = 'password';
      this.error = '';
      this.$nextTick(() => {
        document.getElementById('term-login-password')?.focus({ preventScroll: true });
      });
    },

    resetLoginStep() {
      this.loginStep = 'username';
      this.loginForm.password = '';
      this.error = '';
      this.$nextTick(() => {
        document.getElementById('term-login-username')?.focus({ preventScroll: true });
      });
    },

    async installApp() {
      const prompt = window.slatePwa?.installPrompt;
      if (prompt) {
        await prompt.prompt();
        await prompt.userChoice;
        window.slatePwa.installPrompt = null;
      } else {
        this.showInstallHelp = true;
      }
    },

    applyAppUpdate() {
      if (!this.sessions.length) window.slatePwa?.applyUpdate();
    },

    openMobilePanel(panel) {
      if (panel === 'terminal') {
        this.showMobileMenu = false;
        this.showMobileTools = false;
        this.showMobileSftp = false;
        this.showMobileStatus = false;
        this.terminals[this.activeSessionId]?.focus();
        return;
      }
      this.terminals[this.activeSessionId]?.blur();
      this.showMobileMenu = panel === 'hosts';
      this.showMobileTools = panel === 'tools';
      this.showMobileSftp = panel === 'files';
      this.showMobileStatus = panel === 'status';
      if (panel === 'files') this.rightTab = 'files';
      if (panel === 'status') this.rightTab = 'status';
      if (panel === 'hosts' && !this.isMobile) this.showConnectionManager = true;
    },

    syncDialog(el, visible) {
      if (!!el._dialogOpen === !!visible) return;
      el._dialogOpen = !!visible;
      if (visible) {
        el._returnFocus = document.activeElement;
        this.$nextTick(() => {
          if (!el.contains(document.activeElement)) el.querySelector('button, input, textarea, select')?.focus({ preventScroll: true });
        });
      } else if (el._returnFocus?.isConnected) {
        const target = el._returnFocus;
        if (!(this.isMobile && target.matches('input, textarea'))) target.focus({ preventScroll: true });
      }
    },

    setupViewport() {
      const mobile = window.matchMedia('(max-width: 1023px), (pointer: coarse) and (max-height: 500px)');
      const update = () => {
        cancelAnimationFrame(this._viewportFrame);
        this._viewportFrame = requestAnimationFrame(() => {
          const vv = window.visualViewport;
          if (vv && Math.abs(vv.scale - 1) > 0.02) return;
          this.isMobile = mobile.matches;
          const height = vv?.height || window.innerHeight;
          const offset = vv?.offsetTop || 0;
          const focusedInput = document.activeElement?.matches('input, textarea, [contenteditable="true"]');
          const keyboard = this.isMobile && (
            (window.innerHeight - height > 75) ||
            (!!focusedInput && (window.innerHeight - height > 50 || (window.screen?.height && window.screen.height - height > 160)))
          );
          const root = document.documentElement.style;
          root.setProperty('--vv-height', `${height}px`);
          root.setProperty('--vv-top', `${offset}px`);
          const keyboardStateChanged = this.isKeyboardOpen !== keyboard;
          this.isKeyboardOpen = keyboard;
          clearTimeout(this._resizeTimer);
          this._resizeTimer = setTimeout(() => {
            this.refreshActiveViewport();
            setTimeout(() => this.refreshActiveViewport(), 80);
            setTimeout(() => this.refreshActiveViewport(), 220);
            this.clampEditorWindow();
            const input = document.activeElement;
            if (keyboard && input?.matches('input, textarea') && !input.closest('.xterm, .monaco-editor')) {
              const rect = input.getBoundingClientRect();
              if (rect.bottom > offset + height - 12 || rect.top < offset + 12) input.scrollIntoView({ block: 'nearest' });
            }
          }, 40);
        });
      };
      window.addEventListener('resize', update);
      window.visualViewport?.addEventListener('resize', update);
      window.visualViewport?.addEventListener('scroll', update);
      document.addEventListener('focusin', update);
      document.addEventListener('focusout', update);
      window.addEventListener('pageshow', update);
      this.$watch('isMobile', value => {
        for (const term of Object.values(this.terminals)) term.options.fontSize = value ? 13 : 14;
        this.showMobileMenu = this.showMobileTools = false;
      });
      update();
    },

    get activeStatus() {
      return this.activeSessionId ? this.statuses[this.activeSessionId] : null;
    },

    get activeEditorTab() {
      return this.editorTabs.find(tab => tab.id === this.activeEditorTabId) || null;
    },

    get filteredConnections() {
      const query = this.connectionQuery.trim().toLocaleLowerCase();
      if (!query) return this.connections;
      return this.connections.filter(connection => {
        const name = String(connection.name || '').toLocaleLowerCase();
        const host = String(connection.host || '').toLocaleLowerCase();
        return name.includes(query) || host.includes(query);
      });
    },

    activeSessionType() {
      if (!this.activeSessionId) return '';
      return this.sessions.find(tab => tab.id === this.activeSessionId)?.type || '';
    },

    isConnectionActive(connection) {
      if (!connection || !this.activeSessionId) return false;
      const active = this.sessions.find(tab => tab.id === this.activeSessionId);
      return Number(active?.connectionId) === Number(connection.id);
    },

    getRdpClientOptions() {
      return {
        width: Math.max(1, this.rdpViewportWidth || 1280),
        height: Math.max(1, this.rdpViewportHeight || 720),
        dpi: 96,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai'
      };
    },

    fitRdp(id) {
      const client = this.rdpClients[id];
      const el = document.getElementById(`rdp-${id}`);
      if (!client || !el) return;
      const rect = el.getBoundingClientRect();
      const width = Math.max(1, Math.floor(rect.width || el.clientWidth || 1280));
      const height = Math.max(1, Math.floor(rect.height || el.clientHeight || 720));
      this.rdpViewportWidth = width;
      this.rdpViewportHeight = height;
      try {
        const display = client.getDisplay();
        const displayWidth = display.getWidth?.() || width;
        const displayHeight = display.getHeight?.() || height;
        display.scale(Math.min(width / displayWidth, height / displayHeight) || 1);
        client.sendSize(width, height);
      } catch (_) {}
    },

    resizeActiveTerminal(id = this.activeSessionId) {
      if (!id || !this.terminals[id]) return;
      this.fitAddons[id]?.fit?.();
      const term = this.terminals[id];
      term.refresh?.(0, term.rows - 1);
      try {
        term.scrollToBottom?.();
      } catch (_) {}
      setTimeout(() => {
        try {
          term.scrollToBottom?.();
        } catch (_) {}
      }, 40);
      if (this.socket?.readyState === WebSocket.OPEN) {
        this.socket.send(JSON.stringify({ type: 'ssh:resize', sessionId: id, payload: { cols: term.cols, rows: term.rows } }));
      }
    },

    refreshActiveViewport() {
      this.$nextTick(() => {
        if (!this.activeSessionId) return;
        if (this.activeSessionType() === 'RDP') {
          this.fitRdp(this.activeSessionId);
        } else {
          this.resizeActiveTerminal(this.activeSessionId);
        }
      });
    },

    toggleSidebarPanel() {
      this.showSidebar = !this.showSidebar;
      this.$nextTick(() => {
        this.refreshActiveViewport();
        setTimeout(() => this.refreshActiveViewport(), 80);
      });
    },

    toggleUtilityPanel() {
      if (this.showSftpWidget || this.showStatusWidget) {
        this.showSftpWidget = false;
        this.showStatusWidget = false;
      } else if (this.rightTab === 'status') {
        this.showStatusWidget = true;
      } else {
        this.showSftpWidget = true;
      }
      this.$nextTick(() => {
        this.refreshActiveViewport();
        setTimeout(() => this.refreshActiveViewport(), 80);
      });
    },

    get pathSegments() {
      const current = this.activePath || '.';
      const normalized = current.replace(/\\/g, '/');
      if (normalized === '.' || normalized === './') return [{ label: '.', fullPath: '.' }];
      if (normalized === '/') return [{ label: '/', fullPath: '/' }];
      if (normalized.startsWith('/')) {
        const parts = normalized.split('/').filter(Boolean);
        const result = [{ label: '/', fullPath: '/' }];
        let acc = '';
        for (const part of parts) {
          acc = `${acc}/${part}`;
          result.push({ label: part, fullPath: acc });
        }
        return result;
      }
      const clean = normalized.replace(/^\.\//, '').replace(/^\//, '');
      if (!clean) return [{ label: '/', fullPath: '/' }];
      const parts = clean.split('/').filter(Boolean);
      const result = [{ label: '.', fullPath: '.' }];
      let acc = '.';
      for (const part of parts) {
        acc = acc === '.' ? `./${part}` : `${acc}/${part}`;
        result.push({ label: part, fullPath: acc });
      }
      return result;
    },

    async init() {
      this._resizeTimer = null;
      this._focusTimer = null;
      this.setupViewport();
      this.applyThemeAttrs();

      window.addEventListener('online', () => { this.isOnline = true; });
      window.addEventListener('offline', () => { this.isOnline = false; });
      window.addEventListener('slate:update-ready', () => { this.updateAvailable = true; });
      if (window.slatePwa?.hasUpdate) {
        this.updateAvailable = true;
      }

      // Prevent page bounce / elastic scrolling on iOS Safari while preserving inner scrollables
      document.addEventListener('touchmove', (e) => {
        let target = e.target;
        let isScrollable = false;
        while (target && target !== document.body && target !== document.documentElement) {
          const style = window.getComputedStyle(target);
          const overflowY = style.overflowY;
          const overflowX = style.overflowX;
          const canScrollY = (overflowY === 'auto' || overflowY === 'scroll') && (target.scrollHeight > target.clientHeight);
          const canScrollX = (overflowX === 'auto' || overflowX === 'scroll') && (target.scrollWidth > target.clientWidth);

          if (canScrollY || canScrollX ||
              target.classList.contains('xterm-viewport') ||
              target.classList.contains('monaco-scrollable-element') ||
              target.closest('.xterm-viewport') ||
              target.closest('.monaco-scrollable-element') ||
              target.closest('.terminal-stage') ||
              target.closest('.monaco-editor') ||
              target.closest('#monaco-editor-container') ||
              target.closest('.file-list') ||
              target.closest('.file-panel') ||
              target.closest('.status-panel') ||
              target.closest('.connection-layout') ||
              target.closest('.connection-panel') ||
              target.closest('.terminal-commandbar') ||
              target.closest('.terminal-mobile-bar') ||
              target.closest('.terminal-touch-menu') ||
              target.closest('.terminal-paste-popover') ||
              target.closest('.editor-window') ||
              target.closest('.mobile-keyboard') ||
              target.closest('.tabbar') ||
              target.closest('.term-page') ||
              target.closest('.auth-console') ||
              target.closest('.centered-launcher-wrap') ||
              target.closest('.modal-backdrop')) {
            isScrollable = true;
            break;
          }
          target = target.parentNode;
        }
        if (!isScrollable && e.cancelable) {
          e.preventDefault();
        }
      }, { passive: false });

      document.addEventListener('click', (e) => {
        this.hideContextMenu();
        if (!this.authenticated && !this.booting && !this.needsSetup) {
          const target = e.target;
          if (target && target.tagName !== 'BUTTON' && target.tagName !== 'INPUT' && target.type !== 'checkbox' && !target.closest('button') && !target.closest('label') && !target.closest('.m3-dialog')) {
            this.focusLoginInput(true);
          }
        }
      });
      document.addEventListener('keydown', (e) => {
        if (this.isFullscreen && (e.key === 'Escape' || e.key === 'F11')) {
          this.toggleFullscreen();
          e.preventDefault();
          return;
        }
        if (!this.authenticated && !this.booting && !this.needsSetup) {
          const activeTag = document.activeElement?.tagName;
          const targetId = this.loginStep === 'password' ? 'term-login-password' : 'term-login-username';
          const inputEl = document.getElementById(targetId);
          if (inputEl && document.activeElement !== inputEl && activeTag !== 'INPUT') {
            if (e.key !== 'Tab' && e.key !== 'F12' && e.key !== 'Escape' && !e.ctrlKey && !e.metaKey && !e.altKey) {
              inputEl.focus();
            }
          }
        }
      });
      document.addEventListener('pointermove', (event) => {
        this.dragEditor(event);
        this.resizeEditor(event);
      });
      document.addEventListener('pointerup', () => {
        this.stopEditorDrag();
        this.stopEditorResize();
      });

      // Start global animated spinner ticker (80ms interval)
      if (this.spinnerTimer) clearInterval(this.spinnerTimer);
      this.spinnerTimer = setInterval(() => {
        this.spinnerIndex = (this.spinnerIndex + 1) % this.spinnerFrames.length;
        this.currentSpinner = this.spinnerFrames[this.spinnerIndex];
      }, 80);

      try {
        const needsSetupResp = await fetch('/api/v1/auth/needs-setup');
        this.needsSetup = (await needsSetupResp.json()).needsSetup;
        if (this.needsSetup) {
          this.authenticated = false;
          localStorage.removeItem('slate_auth');
        } else {
          const authResp = await fetch('/api/v1/auth/status');
          const authData = await authResp.json();
          this.authenticated = !!authData.isAuthenticated;
          this.currentUser = authData.user || null;
          if (this.authenticated) {
            localStorage.setItem('slate_auth', 'true');
            await this.loadSettings();
            await this.refreshConnections();
            this.connectSocket().catch(() => {});
          } else {
            localStorage.removeItem('slate_auth');
          }
        }
      } catch (error) {
        this.error = error.message || String(error);
      } finally {
        this.booting = false;
        if (!this.authenticated) {
          this.focusLoginInput();
        }
      }
    },

    async setup() {
      this.error = '';
      const resp = await fetch('/api/v1/auth/setup', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(this.setupForm) });
      const data = await resp.json();
      if (!resp.ok) { this.error = data.message || 'Initialization failed / 初始化失败'; return; }
      this.needsSetup = false;
    },

    async login() {
      this.error = '';
      const username = (this.loginForm.username || '').trim();
      if (!username) {
        this.loginStep = 'username';
        this.focusLoginInput(true);
        return;
      }
      this.authSubmitting = true;
      try {
        const resp = await fetch('/api/v1/auth/login', { 
          method: 'POST', 
          headers: { 'Content-Type': 'application/json' }, 
          body: JSON.stringify(this.loginForm) 
        });
        const data = await resp.json();
        if (!resp.ok) { 
          this.error = 'Login incorrect'; 
          this.loginForm.password = '';
          this.loginStep = 'username';
          this.focusLoginInput(true);
          return; 
        }
        this.authenticated = true;
        this.currentUser = data.user;
        localStorage.setItem('slate_auth', 'true');
        await this.loadSettings();
        await this.refreshConnections();
        this.connectSocket().catch(() => {});
      } catch (err) {
        this.error = 'Login error: network communication failure';
        this.loginStep = 'username';
        this.focusLoginInput(true);
      } finally {
        this.authSubmitting = false;
      }
    },

    async logout() {
      try {
        localStorage.removeItem('slate_auth');
        localStorage.removeItem('slate_connections_cache');
        await fetch('/api/v1/auth/logout', { method: 'POST' });
      } catch (_) {}
      this.authenticated = false;
      this.sessions = [];
      this.activeSessionId = null;
      window.location.reload();
    },

    async loadSettings() {
      const resp = await fetch('/api/v1/settings');
      const settings = await resp.json();
      this.showStatusWidget = settings.showServerStatusWidget !== 'false';
      this.showSftpWidget = settings.showSftpWidget !== 'false';
    },

    async saveSettings() {
      await fetch('/api/v1/settings', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ showServerStatusWidget: String(this.showStatusWidget), showSftpWidget: String(this.showSftpWidget) }) });
    },

    setPalette(name) {
      this.palette = name;
      localStorage.setItem('slatessh_palette', name);
      this.applyThemeAttrs();
    },

    setColorMode(mode) {
      this.colorMode = mode;
      this.theme = mode;
      localStorage.setItem('slatessh_color_mode', mode);
      localStorage.setItem('slatessh-theme', mode);
      this.applyThemeAttrs();
      setTimeout(() => {
        for (const id of Object.keys(this.fitAddons)) this.resizeActiveTerminal(id);
      }, 50);
    },

    toggleColorMode() {
      this.setColorMode(this.colorMode === 'dark' ? 'light' : 'dark');
    },

    toggleTheme() {
      this.toggleColorMode();
    },

    applyThemeAttrs() {
      const root = document.documentElement;
      root.dataset.palette = this.palette;
      root.dataset.theme = this.colorMode;
      const themeColorMeta = document.querySelector('meta[name="theme-color"]');
      if (themeColorMeta) {
        themeColorMeta.setAttribute('content', this.colorMode === 'dark' ? '#10131a' : '#f8f9fc');
      }
    },

    openConnectionManager() {
      this.showConnectionManager = true;
      this.refreshConnections();
    },

    closeConnectionManager() {
      this.showConnectionManager = false;
      this.showConnectionForm = false;
      this.connectionFormSource = null;
      this.testMessage = '';
    },

    async refreshConnections() {
      const resp = await fetch('/api/v1/connections');
      if (!resp.ok) return;
      const data = await resp.json();
      this.connections = data.map(c => {
        const existing = this.connections.find(old => old.id === c.id);
        return { ...c, connecting: existing?.connecting || false };
      });
      try {
        localStorage.setItem('slate_connections_cache', JSON.stringify(this.connections));
      } catch (_) {}
    },

    newConnection(type = 'SSH', source = 'launcher') {
      this.connectionForm = { id: null, name: '', type, host: '', port: type === 'RDP' ? 3389 : 22, username: type === 'RDP' ? '' : 'root', auth_method: 'password', password: '', private_key: '', passphrase: '', notes: '' };
      this.testMessage = '';
      this.testMessageType = 'info';
      this.connectionFormSource = source;
      this.showConnectionForm = true;
      this.showConnectionManager = true;
    },

    cancelConnectionForm() {
      this.showConnectionForm = false;
      this.testMessage = '';
      const returnToDirectory = this.connectionFormSource === 'directory';
      this.connectionFormSource = null;
      if (returnToDirectory) {
        this.showConnectionManager = true;
        return;
      }
      this.closeConnectionManager();
    },

    normalizeConnectionForm() {
      if (this.connectionForm.type === 'RDP') {
        if (!this.connectionForm.port || this.connectionForm.port === 22) this.connectionForm.port = 3389;
        this.connectionForm.auth_method = 'password';
        this.connectionForm.private_key = '';
        this.connectionForm.passphrase = '';
        return;
      }
      if (!this.connectionForm.port || this.connectionForm.port === 3389) this.connectionForm.port = 22;
      if (!this.connectionForm.username) this.connectionForm.username = 'root';
      if (!this.connectionForm.auth_method) this.connectionForm.auth_method = 'password';
    },

    editConnection(connection, source = 'launcher') {
      this.connectionFormSource = source;
      this.showConnectionForm = true;
      this.showConnectionManager = true;
      this.connectionForm = {
        id: connection.id,
        name: connection.name || '',
        type: connection.type || 'SSH',
        host: connection.host || '',
        port: connection.port || ((connection.type || 'SSH') === 'RDP' ? 3389 : 22),
        username: connection.username || ((connection.type || 'SSH') === 'RDP' ? '' : 'root'),
        auth_method: connection.auth_method || 'password',
        password: '',
        private_key: '',
        passphrase: '',
        notes: connection.notes || ''
      };
      this.testMessage = '';
    },

    async saveConnection() {
      const payload = { ...this.connectionForm };
      const method = payload.id ? 'PUT' : 'POST';
      const url = payload.id ? `/api/v1/connections/${payload.id}` : '/api/v1/connections';
      const resp = await fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) });
      const data = await resp.json();
      if (!resp.ok) { this.error = data.message || '保存失败'; return; }
      const returnToDirectory = this.connectionFormSource === 'directory';
      this.showConnectionForm = false;
      this.connectionFormSource = null;
      if (returnToDirectory) {
        this.showConnectionManager = true;
      } else {
        this.showConnectionManager = false;
      }
      this.testMessage = payload.id ? '连接已更新。' : '连接已保存。';
      this.testMessageType = 'success';
      this.connectionForm = { id: null, name: '', type: 'SSH', host: '', port: 22, username: 'root', auth_method: 'password', password: '', private_key: '', passphrase: '', notes: '' };
      await this.refreshConnections();
    },

    async testConnection() {
      this.testResultModal = { visible: true, title: '测试连接', message: '正在测试当前信息，请稍候...', type: 'info' };
      const form = this.connectionForm || {};
      const isSaved = !!form.id;
      const secretsMissing = form.auth_method === 'key'
        ? !(form.private_key && String(form.private_key).trim())
        : !(form.password && String(form.password).length);
      let resp;
      if (isSaved && secretsMissing) {
        resp = await fetch(`/api/v1/connections/${form.id}/test`, { method: 'POST' });
      } else {
        resp = await fetch('/api/v1/connections/test-unsaved', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(form)
        });
      }
      const data = await resp.json();
      this.testResultModal.message = resp.ok ? `连接成功，延迟 ${data.latency} ms` : (data.message || '连接失败');
      this.testResultModal.type = resp.ok ? 'success' : 'error';
    },

    async testSavedConnection(id) {
      this.testResultModal = { visible: true, title: '测试连接', message: '正在测试已保存连接，请稍候...', type: 'info' };
      const resp = await fetch(`/api/v1/connections/${id}/test`, { method: 'POST' });
      const data = await resp.json();
      this.testResultModal.message = resp.ok ? `连接成功，延迟 ${data.latency} ms` : (data.message || '连接失败');
      this.testResultModal.type = resp.ok ? 'success' : 'error';
    },

    async deleteConnection(id) {
      if (!confirm('确定删除这个连接吗？')) return;
      await fetch(`/api/v1/connections/${id}`, { method: 'DELETE' });
      await this.refreshConnections();
    },

    connectSocket() {
      if (this.socket?.readyState === WebSocket.OPEN) return Promise.resolve();
      if (this.socket?.readyState === WebSocket.CONNECTING && this.socketOpenPromise) return this.socketOpenPromise;
      const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
      this.socket = new WebSocket(`${protocol}//${location.host}/ws`);
      this.socketOpenPromise = new Promise((resolve, reject) => {
        this.socket.addEventListener('open', () => {
          this.socketOpenPromise = null;
          resolve();
        }, { once: true });
        this.socket.addEventListener('error', () => {
          this.socketOpenPromise = null;
          reject(new Error('WebSocket 连接失败，请确认服务已启动。'));
        }, { once: true });
        this.socket.addEventListener('close', () => {
          this.socketOpenPromise = null;
          reject(new Error('WebSocket 已断开，请重新连接。'));
        }, { once: true });
      });
      this.socket.addEventListener('message', (event) => {
        const message = JSON.parse(event.data);
        this.handleSocketMessage(message);
      });
      this.socket.addEventListener('close', () => {
        this.socketOpenPromise = null;
        for (const tab of this.sessions) tab.connected = false;
      });
      return this.socketOpenPromise;
    },

    async ensureSocket() {
      if (this.socket?.readyState === WebSocket.OPEN) return true;
      try {
        await this.connectSocket();
        return this.socket?.readyState === WebSocket.OPEN;
      } catch (error) {
        this.testMessage = error.message || 'WebSocket 连接失败';
        this.testMessageType = 'error';
        return false;
      }
    },

    async sendSocket(message) {
      if (!(await this.ensureSocket())) return false;
      try {
        this.socket.send(JSON.stringify(message));
        return true;
      } catch (error) {
        this.testMessage = error.message || '发送指令失败';
        this.testMessageType = 'error';
        return false;
      }
    },

    handleSocketMessage(message) {
      if (message.type === 'error') {
        this.testMessage = message.payload?.message || '操作失败';
        this.testMessageType = 'error';
        const connId = Number(message.payload?.connectionId);
        if (connId) {
          const connection = this.connections.find(c => c.id === connId);
          if (connection) connection.connecting = false;
          const pendingTab = this.sessions.find(x => x.connectionId === connId && x.connecting);
          if (pendingTab) {
            pendingTab.connecting = false;
            pendingTab.error = message.payload?.message || '连接失败';
          }
        } else {
          this.connections.forEach(c => c.connecting = false);
          this.sessions.filter(s => s.connecting).forEach(s => {
            s.connecting = false;
            s.error = message.payload?.message || '连接失败';
          });
        }
      }
      if (message.type === 'status_update') {
        this.statuses = { ...this.statuses, [message.sessionId]: message.payload || {} };
      }
      if (message.type === 'status_error') {
        this.testMessage = message.payload?.message || '状态获取失败';
        this.testMessageType = 'error';
      }
      if (message.type === 'ssh:connected') {
        const connId = Number(message.payload.connectionId);
        const realSessionId = message.sessionId || message.payload.sessionId;
        const connection = this.connections.find(c => c.id === connId);
        if (connection) {
          connection.connecting = false;
        }

        let tab = this.sessions.find(x => (x.connectionId === connId && x.connecting) || x.id === realSessionId);
        if (tab) {
          const oldId = tab.id;
          tab.id = realSessionId;
          tab.connected = true;
          tab.connecting = false;
          tab.error = null;
          if (this.activeSessionId === oldId) {
            this.activeSessionId = realSessionId;
          }
        } else {
          tab = {
            id: realSessionId,
            name: connection?.name || 'SSH',
            connectionId: connId,
            type: 'SSH',
            connected: true,
            connecting: false,
            error: null
          };
          this.sessions.push(tab);
          this.activeSessionId = tab.id;
        }
        this.activePath = '.';
        this.files = [];
        this.pendingPasteTarget = null;
        this.$nextTick(() => {
          this.mountTerminal(tab.id);
        });
        setTimeout(() => this.refreshFiles(), 80);
      }

      if (message.type === 'ssh:output') {
        const term = this.terminals[message.sessionId];
        if (!term) return;
        const data = message.payload?.data || '';
        const encoding = message.payload?.encoding;
        if (encoding === 'base64') {
          const binary = atob(data);
          const bytes = new Uint8Array(binary.length);
          for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
          term.write(bytes, () => {
            if (this.isMobile && this.isKeyboardOpen) {
              try { term.scrollToBottom(); } catch (_) {}
            }
          });
        } else {
          term.write(data, () => {
            if (this.isMobile && this.isKeyboardOpen) {
              try { term.scrollToBottom(); } catch (_) {}
            }
          });
        }
      }

      if (message.type === 'ssh:disconnected') {
        const reason = message.payload?.reason || 'disconnected';
        const autoCloseTab = !!message.payload?.autoCloseTab;
        const term = this.terminals[message.sessionId];
        const tab = this.sessions.find(x => x.id === message.sessionId);
        if (tab) {
          tab.connected = false;
          const connection = this.connections.find(c => c.id === tab.connectionId);
          if (connection) connection.connecting = false;
        }
        if (term) term.writeln(`\r\n[${reason}]`);
        if (autoCloseTab) this.closeTab(message.sessionId, false);
      }

      if (message.type === 'sftp:readdir:result' && message.sessionId === this.activeSessionId) {
        this.activePath = message.payload?.path || '.';
        this.pathInput = this.activePath;
        this.addToPathHistory(this.activePath);
        let entries = message.payload?.entries || [];
        entries.sort((a, b) => {
          if (a.isDir && !b.isDir) return -1;
          if (!a.isDir && b.isDir) return 1;
          return (a.filename || '').localeCompare(b.filename || '');
        });
        this.files = entries;
      }
      if (message.type === 'sftp:writefile:result') {
        const savedPath = message.payload?.path || '';
        this.testMessage = `已保存 ${savedPath}`;
        this.testMessageType = 'success';
        this.playEditorSaveSuccess(savedPath);
      }
      if (message.type === 'sftp:mkdir:result') {
        this.testMessage = `已创建目录 ${message.payload?.path || ''}`;
        this.testMessageType = 'success';
      }
      if (message.type === 'sftp:rename:result') {
        this.testMessage = `已重命名为 ${message.payload?.newPath || ''}`;
        this.testMessageType = 'success';
      }
      if (message.type === 'sftp:unlink:result' || message.type === 'sftp:rmdir:result') {
        this.testMessage = `已删除 ${message.payload?.path || ''}`;
        this.testMessageType = 'success';
      }
      if (message.type === 'sftp:readfile:blocked') {
        this.testMessage = `${message.payload?.reason || '该文件不能在页面中直接编辑'}`;
        this.testMessageType = 'error';
      }
      if (message.type === 'sftp:readfile:result') {
        if (this.pendingPasteTarget && this.pendingPasteTarget.sourcePath === message.payload?.path) {
          this.sendSocket({ type: 'sftp:writefile', sessionId: this.activeSessionId, payload: { path: this.pendingPasteTarget.destinationPath, content: message.payload?.content || '' } });
          this.pendingPasteTarget = null;
          setTimeout(() => this.refreshFiles(), 200);
          return;
        }
        this.openEditorTab(message.payload?.path || 'file', message.payload?.content || '');
      }
    },

    getConnectingElapsed(tab) {
      if (!tab.connectingStart) return '0.0s';
      const sec = ((Date.now() - tab.connectingStart) / 1000).toFixed(1);
      return `${sec}s`;
    },

    getConnectingStep(tab) {
      if (!tab.connectingStart) return 0;
      const elapsed = Date.now() - tab.connectingStart;
      if (elapsed < 600) return 0;
      if (elapsed < 1400) return 1;
      if (elapsed < 2600) return 2;
      return 3;
    },

    getConnectingStatusText(tab) {
      const step = this.getConnectingStep(tab);
      if (step === 0) return 'Resolving host address & initializing socket...';
      if (step === 1) return 'TCP SYN-ACK handshake to port ' + (tab.port || 22) + '...';
      if (step === 2) return 'SSH-2.0 protocol negotiation & Diffie-Hellman key exchange...';
      return 'Authenticating credentials & allocating interactive PTY...';
    },

    async retryConnection(tab) {
      tab.connecting = true;
      tab.error = null;
      tab.connectingStart = Date.now();
      const connection = this.connections.find(c => c.id === tab.connectionId);
      if (connection) connection.connecting = true;
      this.testMessage = `正在重新连接 ${tab.name}...`;
      this.testMessageType = 'info';
      await this.sendSocket({ type: 'ssh:connect', payload: { connectionId: tab.connectionId } });
    },

    cancelConnectingSession(tab) {
      this.closeTab(tab.id);
    },

    async openSession(connection) {
      if ((connection.type || 'SSH') === 'RDP') {
        await this.openRdpSession(connection);
        return;
      }

      const existingConnected = this.sessions.find(tab => tab.connectionId === connection.id && tab.connected);
      if (existingConnected) {
        this.activateSession(existingConnected.id);
        return;
      }

      const existingConnecting = this.sessions.find(tab => tab.connectionId === connection.id && tab.connecting);
      if (existingConnecting) {
        this.activateSession(existingConnecting.id);
        return;
      }

      connection.connecting = true;
      const tempId = 'conn-' + connection.id + '-' + Date.now();
      const tab = {
        id: tempId,
        tempId: tempId,
        connectionId: connection.id,
        name: connection.name || connection.host || 'SSH',
        type: 'SSH',
        connecting: true,
        connected: false,
        connectingStart: Date.now(),
        host: connection.host,
        port: connection.port || 22,
        username: connection.username || 'root',
        error: null
      };
      this.sessions.push(tab);
      this.activeSessionId = tempId;
      this.testMessage = `正在连接 ${connection.name || connection.host}...`;
      this.testMessageType = 'info';
      await this.sendSocket({ type: 'ssh:connect', payload: { connectionId: connection.id } });
    },

    async openRdpSession(connection) {
      if (!window.Guacamole) {
        this.testMessage = 'Guacamole 客户端加载失败，请检查网络或静态资源。';
        this.testMessageType = 'error';
        return;
      }
      const existing = this.sessions.find(tab => tab.connectionId === connection.id && tab.type === 'RDP');
      if (existing) {
        this.activateSession(existing.id);
        return;
      }
      connection.connecting = true;
      const id = crypto.randomUUID();
      const tab = { id, connectionId: connection.id, name: connection.name || connection.host || 'RDP', type: 'RDP', connected: false };
      this.sessions.push(tab);
      this.activeSessionId = id;
      this.files = [];
      this.activePath = '.';
      this.testMessage = `正在打开 RDP 桌面 ${tab.name}...`;
      this.testMessageType = 'info';
      await new Promise(resolve => this.$nextTick(resolve));
      const el = document.getElementById(`rdp-${id}`);
      if (!el) {
        connection.connecting = false;
        this.testMessage = 'RDP 显示区域未初始化。';
        this.testMessageType = 'error';
        return;
      }
      const rect = el.getBoundingClientRect();
      const width = Math.max(1, Math.floor(rect.width || 1280));
      const height = Math.max(1, Math.floor(rect.height || 720));
      const query = new URLSearchParams({ width: String(width), height: String(height), dpi: '96', timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai' });
      const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
      const tunnel = new Guacamole.WebSocketTunnel(`${protocol}//${location.host}/api/v1/rdp/${connection.id}/tunnel?${query.toString()}`);
      const client = new Guacamole.Client(tunnel);
      const guacDisplay = client.getDisplay();
      const display = guacDisplay.getElement();
      el.innerHTML = '';
      el.appendChild(display);
      guacDisplay.onresize = () => this.fitRdp(id);
      this.rdpClients[id] = client;
      client.onstatechange = (state) => {
        tab.connected = state === Guacamole.Client.State.CONNECTED || state === Guacamole.Client.State.WAITING;
        if (state === Guacamole.Client.State.CONNECTED) {
          connection.connecting = false;
          this.testMessage = `RDP 桌面已连接：${tab.name}`;
          this.testMessageType = 'success';
          setTimeout(() => this.fitRdp(id), 50);
        }
        if (state === Guacamole.Client.State.DISCONNECTED) {
          tab.connected = false;
          connection.connecting = false;
          if (this.activeSessionId === id && this.testMessageType !== 'error') {
            this.testMessage = `RDP 桌面已断开：${tab.name}`;
            this.testMessageType = 'error';
          }
        }
      };
      client.onerror = (status) => {
        tab.connected = false;
        connection.connecting = false;
        this.testMessage = status?.message || 'RDP 连接失败。';
        this.testMessageType = 'error';
      };
      const mouse = new Guacamole.Mouse(el);
      mouse.onmousedown = mouse.onmouseup = mouse.onmousemove = (mouseState) => client.sendMouseState(mouseState);
      const keyboard = new Guacamole.Keyboard(document);
      keyboard.onkeydown = (keysym) => {
        if (this.activeSessionId !== id) return true;
        client.sendKeyEvent(1, keysym);
        return false;
      };
      keyboard.onkeyup = (keysym) => {
        if (this.activeSessionId !== id) return true;
        client.sendKeyEvent(0, keysym);
        return false;
      };
      this.rdpKeyboards[id] = keyboard;
      el.tabIndex = 0;
      el.addEventListener('click', () => el.focus());
      client.connect();
      setTimeout(() => this.fitRdp(id), 100);
    },

    activateSession(id) {
      this.activeSessionId = id;
      if (this.activeSessionType() !== 'RDP') this.refreshFiles();
      if (this.rdpClients[id]) this.fitRdp(id);
      if (this.terminals[id]) {
        this.$nextTick(() => {
          setTimeout(() => {
            this.resizeActiveTerminal(id);
            this.terminals[id]?.focus();
          }, 0);
        });
      }
    },

    closeTab(id, notify = true) {
      const idx = this.sessions.findIndex(x => x.id === id);
      if (idx >= 0) {
        const tab = this.sessions[idx];
        const connection = this.connections.find(c => c.id === tab.connectionId);
        if (connection) connection.connecting = false;
        this.sessions.splice(idx, 1);
      }
      if (notify && this.socket?.readyState === WebSocket.OPEN && this.terminals[id]) {
        this.socket.send(JSON.stringify({ type: 'ssh:disconnect', sessionId: id }));
      }
      if (this.rdpKeyboards[id]) {
        try { this.rdpKeyboards[id].onkeydown = null; this.rdpKeyboards[id].onkeyup = null; } catch (_) {}
        delete this.rdpKeyboards[id];
      }
      if (this.rdpClients[id]) {
        try { this.rdpClients[id].disconnect(); } catch (_) {}
        delete this.rdpClients[id];
      }
      const rdpEl = document.getElementById(`rdp-${id}`);
      if (rdpEl) rdpEl.innerHTML = '';
      if (this.terminals[id]) {
        this.terminals[id].dispose();
        delete this.terminals[id];
        delete this.searchAddons[id];
        delete this.fitAddons[id];
      }
      delete this.statuses[id];
      this.pendingPasteTarget = null;
      if (this.activeSessionId === id) {
        this.activeSessionId = this.sessions[0]?.id || null;
        this.activePath = '.';
        this.files = [];
        if (this.activeSessionId && this.activeSessionType() !== 'RDP') {
          setTimeout(() => this.refreshFiles(), 80);
        }
      }
      if (!this.activeSessionId) {
        this.isFullscreen = false;
        document.body.classList.remove('terminal-web-fullscreen');
      }
      if (this.terminalCopyViewer.visible && this.activeSessionId !== id) this.closeTerminalCopyViewer();
    },

    mountTerminal(id) {
      if (this.terminals[id]) return;
      const el = document.getElementById(`terminal-${id}`);
      if (!el || !window.Terminal) return;
      const isMobileDevice = this.isMobile;
      const term = new window.Terminal({ 
        cursorBlink: true, 
        cursorStyle: 'underline',
        fontSize: isMobileDevice ? 12 : 14, 
        fontFamily: '"Cascadia Code", "JetBrains Mono", "Fira Code", Consolas, "Courier New", monospace',
        theme: { 
          background: '#090a0f',
          foreground: '#e6edf3',
          cursor: '#5af78e',
          cursorAccent: '#090a0f',
          black: '#090a0f',
          red: '#ff7b72',
          green: '#3fb950',
          yellow: '#d29922',
          blue: '#58a6ff',
          magenta: '#bc8cff',
          cyan: '#39c5cf',
          white: '#b1bac4',
          brightBlack: '#484f58',
          brightRed: '#ffa198',
          brightGreen: '#56d364',
          brightYellow: '#e3b341',
          brightBlue: '#79c0ff',
          brightMagenta: '#d2a8ff',
          brightCyan: '#56d4dd',
          brightWhite: '#f0f6fc'
        }, 
        convertEol: true, 
        scrollback: 5000,
        allowTransparency: true,
        allowProposedApi: true,
        rightClickSelectsWord: isMobileDevice,
        scrollOnUserInput: true,
        scrollSensitivity: isMobileDevice ? 2 : 1,
        fastScrollSensitivity: isMobileDevice ? 6 : 5
      });
      const fit = window.FitAddon ? new window.FitAddon.FitAddon() : null;
      const search = window.SearchAddon ? new window.SearchAddon.SearchAddon() : null;
      if (fit) term.loadAddon(fit);
      if (search) term.loadAddon(search);
      term.open(el);
      setTimeout(() => this.resizeActiveTerminal(id), 30);
      setTimeout(() => term.focus(), 60);
      term.onData((data) => {
        let payloadData = data;
        if (this.ctrlKeyActive && data.length === 1) {
          const charCode = data.charCodeAt(0);
          if ((charCode >= 65 && charCode <= 90) || (charCode >= 97 && charCode <= 122)) {
            payloadData = String.fromCharCode(charCode & 0x1F);
          }
          this.ctrlKeyActive = false;
        }
        this.sendSocket({ type: 'ssh:input', sessionId: id, payload: { data: payloadData } });
        if (this.isMobile) {
          try { term.scrollToBottom(); } catch (_) {}
        }
      });
      
      let selecting = false;
      let selectMoved = false;
      let selectStartX = 0;
      let selectStartY = 0;
      let pendingSelection = '';
      if (typeof term.onSelectionChange === 'function') {
        term.onSelectionChange(() => {
          try {
            const t = term.getSelection();
            if (t) pendingSelection = t;
          } catch (_) {}
        });
      }
      const copyTerminalDragSelection = () => {
        if (!this.terminals[id]) return false;
        const text = (term.getSelection && term.getSelection()) || pendingSelection || '';
        if (!text) return false;
        if (this.copyTextViaExecCommand(text)) return true;
        try {
          if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
            void navigator.clipboard.writeText(text);
            return true;
          }
        } catch (_) {}
        return false;
      };
      const onDocMouseUp = (event) => {
        if (!selecting) return;
        if (typeof event.button === 'number' && event.button !== 0) return;
        selecting = false;
        document.removeEventListener('mouseup', onDocMouseUp, true);
        document.removeEventListener('mousemove', onDocMouseMove, true);
        if (!selectMoved) return;
        if (copyTerminalDragSelection()) {
          pendingSelection = '';
          return;
        }
        requestAnimationFrame(() => {
          if (copyTerminalDragSelection()) {
            pendingSelection = '';
            return;
          }
          setTimeout(() => {
            copyTerminalDragSelection();
            pendingSelection = '';
          }, 0);
        });
      };
      const onDocMouseMove = (event) => {
        if (!selecting) return;
        if (Math.abs(event.clientX - selectStartX) > 2 || Math.abs(event.clientY - selectStartY) > 2) {
          selectMoved = true;
        }
      };
      term.element?.addEventListener('mousedown', (event) => {
        if (event.button !== 0) return;
        selecting = true;
        selectMoved = false;
        selectStartX = event.clientX;
        selectStartY = event.clientY;
        pendingSelection = '';
        document.addEventListener('mousemove', onDocMouseMove, true);
        document.addEventListener('mouseup', onDocMouseUp, true);
      });

      term.element?.addEventListener('contextmenu', async (event) => {
        event.preventDefault();
        event.stopPropagation();
        try {
          const text = await navigator.clipboard.readText();
          if (text) {
            this.pasteTextToSession(id, text);
          }
        } catch (_) {
          if (!this.isMobile) {
            this.openTerminalPasteBox();
          }
        }
      });
      this.terminals[id] = term;
      this.searchAddons[id] = search;
      this.fitAddons[id] = fit;
      window.addEventListener('resize', () => this.resizeActiveTerminal(id));

      if (this.isMobile) {
        let touchStartY = 0;
        let touchStartX = 0;
        let touchMoved = false;

        el.addEventListener('touchstart', (e) => {
          if (e.touches.length !== 1) return;
          const touch = e.touches[0];
          touchStartY = touch.clientY;
          touchStartX = touch.clientX;
          touchMoved = false;
        }, { passive: true });

        el.addEventListener('touchmove', (e) => {
          if (e.touches.length !== 1) return;
          const touch = e.touches[0];
          const deltaY = touchStartY - touch.clientY;
          const deltaX = touchStartX - touch.clientX;
          const absY = Math.abs(deltaY);
          const absX = Math.abs(deltaX);

          if (absY > 4 || absX > 4) {
            touchMoved = true;
          }
          if (absY > absX && absY > 4) {
            if (e.cancelable) e.preventDefault();
            const lines = Math.trunc(deltaY / 11);
            if (lines !== 0) {
              term.scrollLines(lines);
              touchStartY = touch.clientY;
            }
          }
        }, { passive: false });

        el.addEventListener('touchend', () => {
          if (!touchMoved) {
            setTimeout(() => term.focus(), 0);
          }
        }, { passive: true });
      }
    },

    toggleFullscreen() {
      this.isFullscreen = !this.isFullscreen;
      document.body.classList.toggle('terminal-web-fullscreen', this.isFullscreen);
      this.$nextTick(() => {
        if (!this.activeSessionId) return;
        if (this.activeSessionType() === 'RDP') {
          this.fitRdp(this.activeSessionId);
        } else {
          this.resizeActiveTerminal(this.activeSessionId);
        }
        setTimeout(() => {
          if (this.activeSessionType() === 'RDP') this.fitRdp(this.activeSessionId);
          else this.resizeActiveTerminal(this.activeSessionId);
        }, 180);
      });
    },

    clearActiveTerminal() {
      const term = this.terminals[this.activeSessionId];
      if (term) term.clear();
    },

    searchTerminal() {
      if (!this.activeSessionId || !this.terminalSearch) return;
      this.searchAddons[this.activeSessionId]?.findNext?.(this.terminalSearch, { caseSensitive: false });
    },

    findNextTerminalResult() {
      if (!this.activeSessionId || !this.terminalSearch) return;
      this.searchAddons[this.activeSessionId]?.findNext?.(this.terminalSearch, { caseSensitive: false });
    },

    findPreviousTerminalResult() {
      if (!this.activeSessionId || !this.terminalSearch) return;
      this.searchAddons[this.activeSessionId]?.findPrevious?.(this.terminalSearch, { caseSensitive: false });
    },

    normalizeTerminalPasteText(text) {
      return String(text ?? '')
        .replace(/\r\n/g, '\n')
        .replace(/\r/g, '\n')
        .replace(/\n/g, '\r');
    },

    pasteTextToSession(sessionId, text) {
      if (!sessionId || text == null || text === '') return;
      const term = this.terminals[sessionId];
      if (term && typeof term.paste === 'function') {
        try {
          term.paste(String(text));
          return;
        } catch (_) {}
      }
      const normalized = this.normalizeTerminalPasteText(text);
      this.sendSocket({ type: 'ssh:input', sessionId, payload: { data: normalized } });
    },

    copyTextViaExecCommand(value) {
      let ta;
      try {
        ta = document.createElement('textarea');
        ta.value = value;
        ta.setAttribute('readonly', '');
        ta.style.cssText = 'position:fixed;top:0;left:0;width:2px;height:2px;padding:0;border:0;opacity:0;z-index:2147483647;';
        document.body.appendChild(ta);
        const selection = document.getSelection();
        const previousRange = selection && selection.rangeCount > 0 ? selection.getRangeAt(0) : null;
        ta.focus({ preventScroll: true });
        ta.select();
        ta.setSelectionRange(0, value.length);
        const ok = document.execCommand('copy');
        if (previousRange && selection) {
          selection.removeAllRanges();
          selection.addRange(previousRange);
        }
        return !!ok;
      } catch (_) {
        return false;
      } finally {
        ta?.remove?.();
      }
    },

    async writeClipboardText(text, options = {}) {
      if (text == null || text === '') return false;
      const value = String(text);
      const preferSync = !!(options && options.preferSync);

      if (preferSync) {
        if (this.copyTextViaExecCommand(value)) return true;
        try {
          if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
            await navigator.clipboard.writeText(value);
            return true;
          }
        } catch (_) {}
        return false;
      }

      if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
        try {
          await navigator.clipboard.writeText(value);
          return true;
        } catch (_) {}
      }

      return this.copyTextViaExecCommand(value);
    },

    async copyTerminalSelection() {
      const term = this.terminals[this.activeSessionId];
      const value = term?.getSelection?.() || this.getTerminalAllText(this.activeSessionId);
      if (!value) {
        this.testMessage = '没有可复制的终端内容';
        this.testMessageType = 'info';
        return;
      }
      const ok = await this.writeClipboardText(value);
      this.testMessage = ok ? '已复制到剪贴板' : '复制失败';
      this.testMessageType = ok ? 'success' : 'error';
      this.refocusTerminal();
    },

    getTerminalAllText(id = this.activeSessionId) {
      const term = this.terminals[id];
      const buffer = term?.buffer?.active;
      if (!term || !buffer) return '';
      const lines = [];
      for (let i = 0; i < buffer.length; i++) {
        const line = buffer.getLine(i)?.translateToString(true);
        if (line !== undefined) lines.push(line);
      }
      return lines.join('\n').replace(/\s+$/g, '');
    },

    openTerminalCopyViewer() {
      const text = this.getTerminalAllText(this.activeSessionId);
      this.terminalCopyViewer.text = text || '';
      this.terminalCopyViewer.visible = true;
      this.$nextTick(() => {
        const textarea = document.querySelector('.terminal-copy-viewer textarea');
        if (textarea) {
          textarea.focus();
          textarea.select();
        }
      });
    },

    selectAllCopyViewer() {
      const textarea = document.querySelector('.terminal-copy-viewer textarea');
      if (textarea) {
        textarea.focus();
        textarea.select();
        try {
          textarea.setSelectionRange(0, textarea.value.length);
        } catch (_) {}
      }
    },

    async copyAllFromViewer() {
      const text = this.terminalCopyViewer.text;
      if (!text) return;
      const ok = await this.writeClipboardText(text);
      this.testMessage = ok ? '已复制终端全部内容' : '已全选，请长按文本框复制';
      this.testMessageType = ok ? 'success' : 'info';
    },

    closeTerminalCopyViewer() {
      this.terminalCopyViewer.visible = false;
      this.refocusTerminal(true);
    },

    closeTerminalPasteBox() {
      this.terminalPasteBox.visible = false;
      this.terminalPasteBox.text = '';
      this.$nextTick(() => {
        this.refocusTerminal(true);
      });
    },

    openTerminalPasteBox() {
      this.terminalPasteBox.text = '';
      this.terminalPasteBox.visible = true;
      this.$nextTick(() => {
        const input = document.querySelector('.terminal-paste-popover textarea');
        input?.focus?.();
        setTimeout(() => input?.focus?.(), 80);
      });
    },

    async sendTerminalPasteText() {
      const text = this.terminalPasteBox.text;
      const targetSessionId = this.activeSessionId;
      this.terminalPasteBox.visible = false;
      this.terminalPasteBox.text = '';
      if (text && targetSessionId) {
        this.pasteTextToSession(targetSessionId, text);
      }
      this.$nextTick(() => {
        this.refocusTerminal(true);
      });
    },

    async refreshFiles() {
      if (this.activeSessionType() === 'RDP') {
        this.files = [];
        this.testMessage = 'RDP 会话不提供 SFTP 文件浏览。';
        this.testMessageType = 'info';
        return;
      }
      if (!this.activeSessionId) {
        this.testMessage = '请先连接 SSH 会话。';
        this.testMessageType = 'error';
        return;
      }
      await this.sendSocket({ type: 'sftp:readdir', sessionId: this.activeSessionId, payload: { path: this.activePath || '.' } });
      await this.sendSocket({ type: 'status:refresh', sessionId: this.activeSessionId, payload: {} });
    },

    goParentDir() {
      if (this.activePath === '.' || this.activePath === '/' || !this.activePath) return;
      const normalized = this.activePath.replace(/\\/g, '/').replace(/\/$/, '');
      const slashIndex = normalized.lastIndexOf('/');
      const next = normalized.startsWith('/') ? (slashIndex <= 0 ? '/' : normalized.substring(0, slashIndex)) : (normalized.substring(0, slashIndex) || '.');
      this.activePath = next === '.' ? '.' : next;
      this.refreshFiles();
    },

    jumpToPath(path) {
      this.activePath = path;
      this.refreshFiles();
    },

    addToPathHistory(path) {
      if (!path || path === '.' || path === './') return;
      this.pathHistory = this.pathHistory.filter(p => p !== path);
      this.pathHistory.unshift(path);
      if (this.pathHistory.length > 15) {
        this.pathHistory.pop();
      }
      localStorage.setItem('slatessh-path-history', JSON.stringify(this.pathHistory));
    },

    getPathSuggestions() {
      const query = (this.pathInput || '').trim();
      const suggestions = new Set();

      if (this.activePath) {
        const currentSubdirs = this.files.filter(f => f.isDir).map(f => f.filename);
        for (const sub of currentSubdirs) {
          const full = this.joinRemotePath(this.activePath, sub);
          suggestions.add(full);
        }
      }

      const commonDirs = ['/', '/root', '/home', '/etc', '/var', '/tmp', '/usr/local'];
      for (const dir of commonDirs) {
        suggestions.add(dir);
      }

      for (const path of this.pathHistory) {
        suggestions.add(path);
      }

      let list = Array.from(suggestions);

      if (query) {
        const qLower = query.toLowerCase();
        list = list.filter(item => {
          return item.toLowerCase().includes(qLower) && item !== query;
        });
      }

      return list.slice(0, 8);
    },

    submitPath() {
      const target = this.pathInput ? this.pathInput.trim() : '';
      if (!target) return;
      this.jumpToPath(target);
      this.showPathDropdown = false;
      this.activeSuggestionIndex = -1;
    },

    navigateSuggestions(direction) {
      const list = this.getPathSuggestions();
      if (!list.length) return;
      this.showPathDropdown = true;
      this.activeSuggestionIndex += direction;
      if (this.activeSuggestionIndex < 0) {
        this.activeSuggestionIndex = list.length - 1;
      } else if (this.activeSuggestionIndex >= list.length) {
        this.activeSuggestionIndex = 0;
      }
    },

    selectSuggestionOrSubmit() {
      const list = this.getPathSuggestions();
      if (this.showPathDropdown && this.activeSuggestionIndex >= 0 && this.activeSuggestionIndex < list.length) {
        this.selectPathSuggestion(list[this.activeSuggestionIndex]);
      } else {
        this.submitPath();
      }
    },

    selectPathSuggestion(path) {
      this.pathInput = path;
      this.submitPath();
    },

    openDir(name) {
      this.activePath = this.joinRemotePath(this.activePath, name);
      this.refreshFiles();
    },

    readFile(name) {
      const target = this.joinRemotePath(this.activePath, name);
      this.sendSocket({ type: 'sftp:readfile', sessionId: this.activeSessionId, payload: { path: target } });
    },

    openEditorTab(path, content) {
      const existing = this.editorTabs.find(tab => tab.path === path);
      this.monacoReady = false;
      if (window.appMonacoInstance) {
        window.appMonacoInstance.dispose();
        window.appMonacoInstance = null;
      }
      if (existing) {
        existing.content = content;
        this.activeEditorTabId = existing.id;
        this.placeEditorWindow();
        this.$nextTick(() => this.initMonacoEditor());
        return;
      }
      const tab = { id: crypto.randomUUID(), path, name: path.split('/').pop() || path, content };
      this.editorTabs.push(tab);
      this.activeEditorTabId = tab.id;
      this.placeEditorWindow();
      this.$nextTick(() => this.initMonacoEditor());
    },

    activateEditorTab(id) {
      this.activeEditorTabId = id;
    },

    closeEditorTab(id) {
      const idx = this.editorTabs.findIndex(tab => tab.id === id);
      if (idx >= 0) this.editorTabs.splice(idx, 1);
      if (this.activeEditorTabId === id) this.activeEditorTabId = this.editorTabs[0]?.id || null;
      if (!this.activeEditorTabId) {
        this.monacoReady = false;
        this.editorWindow.maximized = false;
        this.editorWindow.minimized = false;
        this.editorWindow.dragging = false;
        this.editorWindow.resizing = false;
        this.editorWindow.restore = null;
        if (window.appMonacoInstance) {
          window.appMonacoInstance.dispose();
          window.appMonacoInstance = null;
        }
        if (this.isMobile) {
          this.showMobileSftp = true;
        }
      } else {
        this.layoutEditor();
      }
    },

    getMonacoLanguage(filename) {
      if (!filename) return 'plaintext';
      const parts = filename.toLowerCase().split('.');
      if (filename.toLowerCase() === 'dockerfile') return 'dockerfile';
      const ext = parts.pop();
      const map = {
        js: 'javascript', jsx: 'javascript',
        ts: 'typescript', tsx: 'typescript',
        html: 'html', htm: 'html',
        css: 'css',
        json: 'json', jsonc: 'json',
        go: 'go',
        py: 'python',
        sh: 'shell', bash: 'shell', zsh: 'shell',
        md: 'markdown', markdown: 'markdown',
        yaml: 'yaml', yml: 'yaml',
        xml: 'xml', svg: 'xml',
        sql: 'sql',
        java: 'java',
        c: 'c',
        cpp: 'cpp', cc: 'cpp', h: 'cpp', hpp: 'cpp',
        cs: 'csharp',
        php: 'php',
        ps1: 'powershell',
        rs: 'rust',
        rb: 'ruby',
        swift: 'swift',
        kt: 'kotlin', kts: 'kotlin',
        ini: 'ini', conf: 'ini', env: 'ini', properties: 'ini'
      };
      return map[ext] || 'plaintext';
    },

    initMonacoEditor() {
      const container = document.getElementById('monaco-editor-container');
      if (!container || !this.activeEditorTab) return;
      if (!window.require) {
        clearTimeout(this.monacoInitTimer);
        this.monacoInitTimer = setTimeout(() => this.initMonacoEditor(), 80);
        return;
      }

      window.require(['vs/editor/editor.main'], () => {
        const container = document.getElementById('monaco-editor-container');
        if (!container || !this.activeEditorTab) return;

        if (!window.monacoThemesDefined) {
          window.monaco.editor.defineTheme('slatessh-dark', {
            base: 'vs-dark',
            inherit: true,
            rules: [],
            colors: {
              'editor.background': '#0a0c10',
              'editor.lineHighlightBackground': '#161b22',
              'editorCursor.foreground': '#3fb950',
              'editor.selectionBackground': '#264F78',
              'editor.inactiveSelectionBackground': '#264F7844'
            }
          });
          window.monaco.editor.defineTheme('slatessh-light', {
            base: 'vs',
            inherit: true,
            rules: [],
            colors: {
              'editor.background': '#0a0c10',
              'editor.lineHighlightBackground': '#161b22',
              'editorCursor.foreground': '#3fb950',
              'editor.selectionBackground': '#264F78',
              'editor.inactiveSelectionBackground': '#264F7844'
            }
          });
          window.monacoThemesDefined = true;
        }

        if (window.appMonacoInstance) {
          window.appMonacoInstance.dispose();
          window.appMonacoInstance = null;
        }
        window.appMonacoInstance = window.monaco.editor.create(container, {
          value: this.activeEditorTab.content || '',
          language: this.getMonacoLanguage(this.activeEditorTab.name),
          theme: 'slatessh-dark',
          automaticLayout: true,
          minimap: { enabled: false },
          fontSize: 14,
          fontFamily: '"Cascadia Code", "JetBrains Mono", "Fira Code", Consolas, monospace'
        });
        window.appMonacoInstance.onDidChangeModelContent(() => {
          if (this.activeEditorTab && !this.isMonacoUpdating) {
            this.activeEditorTab.content = window.appMonacoInstance.getValue();
          }
        });
        this.monacoReady = true;
        this.$nextTick(() => window.appMonacoInstance?.layout?.());
      });
    },

    updateMonacoEditor() {
      const tab = this.activeEditorTab;
      const tabContent = tab?.content;
      const tabName = tab?.name;

      if (!window.appMonacoInstance) return;
      if (tab) {
        this.isMonacoUpdating = true;
        const currentVal = window.appMonacoInstance.getValue();
        if (currentVal !== tabContent) {
          window.appMonacoInstance.setValue(tabContent || '');
        }
        const lang = this.getMonacoLanguage(tabName);
        window.monaco.editor.setModelLanguage(window.appMonacoInstance.getModel(), lang);
        window.monaco.editor.setTheme('slatessh-dark');
        this.isMonacoUpdating = false;
      }
    },

    placeEditorWindow() {
      const viewportHeight = window.visualViewport?.height || window.innerHeight;
      const width = this.isMobile ? Math.max(320, window.innerWidth - 24) : Math.min(920, Math.max(560, window.innerWidth * 0.68));
      const height = this.isMobile ? Math.max(420, viewportHeight - 96) : Math.min(680, Math.max(420, viewportHeight * 0.72));
      this.editorWindow.width = width;
      this.editorWindow.height = height;
      this.editorWindow.x = Math.max(8, Math.round((window.innerWidth - width) / 2));
      this.editorWindow.y = Math.max(56, Math.round((viewportHeight - height) / 2));
      this.editorWindow.maximized = false;
      this.editorWindow.minimized = false;
      this.editorWindow.restore = null;
      this.layoutEditor();
    },

    editorWindowStyle() {
      const viewportHeight = window.visualViewport?.height || window.innerHeight;
      if (this.editorWindow.maximized) {
        const left = 0;
        const top = 0;
        const width = window.innerWidth;
        const height = viewportHeight;
        return `left:${left}px;top:${top}px;width:${width}px;height:${height}px`;
      }
      if (this.editorWindow.minimized) {
        const width = Math.min(360, Math.max(260, window.innerWidth - 24));
        const left = Math.max(12, window.innerWidth - width - 16);
        const top = Math.max(12, viewportHeight - 56 - 12);
        return `left:${left}px;top:${top}px;width:${width}px;height:44px`;
      }
      return `left:${this.editorWindow.x}px;top:${this.editorWindow.y}px;width:${this.editorWindow.width}px;height:${this.editorWindow.height}px`;
    },

    layoutEditor() {
      this.$nextTick(() => {
        try {
          const inst = window.appMonacoInstance || this.monacoInstance;
          inst?.layout?.();
        } catch (_) {}
      });
    },

    clampEditorWindow() {
      if (!this.activeEditorTab || this.editorWindow.maximized || this.editorWindow.minimized) {
        this.layoutEditor();
        return;
      }
      const viewportHeight = window.visualViewport?.height || window.innerHeight;
      const minWidth = this.isMobile ? 300 : 420;
      const minHeight = this.isMobile ? 280 : 300;
      this.editorWindow.width = Math.min(Math.max(this.editorWindow.width, minWidth), Math.max(minWidth, window.innerWidth - 16));
      this.editorWindow.height = Math.min(Math.max(this.editorWindow.height, minHeight), Math.max(minHeight, viewportHeight - 24));
      this.editorWindow.x = Math.min(Math.max(8, this.editorWindow.x), Math.max(8, window.innerWidth - this.editorWindow.width - 8));
      this.editorWindow.y = Math.min(Math.max(8, this.editorWindow.y), Math.max(8, viewportHeight - this.editorWindow.height - 8));
      this.layoutEditor();
    },

    rememberEditorGeometry() {
      if (this.editorWindow.maximized || this.editorWindow.minimized) return;
      this.editorWindow.restore = {
        x: this.editorWindow.x,
        y: this.editorWindow.y,
        width: this.editorWindow.width,
        height: this.editorWindow.height
      };
    },

    restoreEditorGeometry() {
      const restore = this.editorWindow.restore;
      if (restore) {
        this.editorWindow.x = restore.x;
        this.editorWindow.y = restore.y;
        this.editorWindow.width = restore.width;
        this.editorWindow.height = restore.height;
      } else if (!this.editorWindow.width || !this.editorWindow.height) {
        this.placeEditorWindow();
        return;
      }
      this.editorWindow.maximized = false;
      this.editorWindow.minimized = false;
      this.editorWindow.restore = null;
      this.clampEditorWindow();
      this.$nextTick(() => {
        this.layoutEditor();
        requestAnimationFrame(() => this.layoutEditor());
      });
    },

    minimizeEditor() {
      if (!this.editorWindow.minimized) {
        this.rememberEditorGeometry();
      }
      this.editorWindow.maximized = false;
      this.editorWindow.minimized = !this.editorWindow.minimized;
      if (!this.editorWindow.minimized) {
        this.restoreEditorGeometry();
      }
      this.layoutEditor();
    },

    toggleEditorMaximize() {
      this.editorWindow.maximized = !this.editorWindow.maximized;
      this.editorWindow.minimized = false;
      this.$nextTick(() => {
        this.layoutEditor();
        setTimeout(() => this.layoutEditor(), 60);
        setTimeout(() => this.layoutEditor(), 250);
      });
    },

    startEditorDrag(event) {
      if (event.target.closest('button')) return;
      if (this.editorWindow.minimized) {
        this.minimizeEditor();
        return;
      }
      if (this.editorWindow.maximized) return;
      this.editorWindow.dragging = true;
      this.editorWindow.offsetX = event.clientX - this.editorWindow.x;
      this.editorWindow.offsetY = event.clientY - this.editorWindow.y;
      try { event.currentTarget.setPointerCapture?.(event.pointerId); } catch (_) {}
      event.preventDefault();
    },

    dragEditor(event) {
      if (!this.editorWindow.dragging) return;
      const viewportHeight = window.visualViewport?.height || window.innerHeight;
      this.editorWindow.x = Math.min(Math.max(8, event.clientX - this.editorWindow.offsetX), Math.max(8, window.innerWidth - 120));
      this.editorWindow.y = Math.min(Math.max(8, event.clientY - this.editorWindow.offsetY), Math.max(8, viewportHeight - 56));
    },

    stopEditorDrag() {
      this.editorWindow.dragging = false;
    },

    startEditorResize(event, edge) {
      if (this.editorWindow.maximized || this.editorWindow.minimized) return;
      this.editorWindow.resizing = true;
      this.editorWindow.resizeEdge = edge;
      this.editorWindow.startX = event.clientX;
      this.editorWindow.startY = event.clientY;
      this.editorWindow.startWidth = this.editorWindow.width;
      this.editorWindow.startHeight = this.editorWindow.height;
      this.editorWindow.startLeft = this.editorWindow.x;
      this.editorWindow.startTop = this.editorWindow.y;
      try { event.currentTarget.setPointerCapture?.(event.pointerId); } catch (_) {}
    },

    resizeEditor(event) {
      if (!this.editorWindow.resizing) return;
      const edge = this.editorWindow.resizeEdge || '';
      const dx = event.clientX - this.editorWindow.startX;
      const dy = event.clientY - this.editorWindow.startY;
      const viewportHeight = window.visualViewport?.height || window.innerHeight;
      const minWidth = this.isMobile ? 300 : 420;
      const minHeight = this.isMobile ? 260 : 300;
      let width = this.editorWindow.startWidth;
      let height = this.editorWindow.startHeight;
      let x = this.editorWindow.startLeft;
      let y = this.editorWindow.startTop;

      if (edge.includes('e')) width = this.editorWindow.startWidth + dx;
      if (edge.includes('s')) height = this.editorWindow.startHeight + dy;
      if (edge.includes('w')) {
        width = this.editorWindow.startWidth - dx;
        x = this.editorWindow.startLeft + dx;
      }
      if (edge.includes('n')) {
        height = this.editorWindow.startHeight - dy;
        y = this.editorWindow.startTop + dy;
      }

      if (width < minWidth) {
        if (edge.includes('w')) x -= minWidth - width;
        width = minWidth;
      }
      if (height < minHeight) {
        if (edge.includes('n')) y -= minHeight - height;
        height = minHeight;
      }

      width = Math.min(width, window.innerWidth - 16);
      height = Math.min(height, viewportHeight - 16);
      this.editorWindow.width = width;
      this.editorWindow.height = height;
      this.editorWindow.x = Math.min(Math.max(8, x), Math.max(8, window.innerWidth - width - 8));
      this.editorWindow.y = Math.min(Math.max(8, y), Math.max(8, viewportHeight - height - 8));
      this.layoutEditor();
    },

    stopEditorResize() {
      this.editorWindow.resizing = false;
      this.editorWindow.resizeEdge = '';
    },

    saveEditor() {
      const tab = this.activeEditorTab;
      if (!tab || !this.activeSessionId) return;
      if (this.editorSaveFeedback.state === 'saving') return;
      if (window.appMonacoInstance && !this.isMonacoUpdating) {
        try { tab.content = window.appMonacoInstance.getValue(); } catch (_) {}
      }
      this.setEditorSaveFeedback('saving', tab.path);
      this.sendSocket({
        type: 'sftp:writefile',
        sessionId: this.activeSessionId,
        payload: { path: tab.path, content: tab.content }
      });
      this.testMessage = `正在保存 ${tab.path}`;
      this.testMessageType = 'info';
      setTimeout(() => this.refreshFiles(), 150);
    },

    setEditorSaveFeedback(state, path = '') {
      if (this.editorSaveFeedback.timer) {
        clearTimeout(this.editorSaveFeedback.timer);
        this.editorSaveFeedback.timer = null;
      }
      this.editorSaveFeedback.state = state;
      this.editorSaveFeedback.path = path || '';
      if (state === 'saved') {
        this.editorSaveFeedback.timer = setTimeout(() => {
          this.editorSaveFeedback.state = 'idle';
          this.editorSaveFeedback.path = '';
          this.editorSaveFeedback.timer = null;
        }, 1800);
      } else if (state === 'saving') {
        this.editorSaveFeedback.timer = setTimeout(() => {
          if (this.editorSaveFeedback.state === 'saving') {
            this.editorSaveFeedback.state = 'idle';
            this.editorSaveFeedback.path = '';
            this.editorSaveFeedback.timer = null;
          }
        }, 8000);
      }
    },

    playEditorSaveSuccess(path = '') {
      const activePath = this.activeEditorTab?.path || '';
      const feedbackPath = this.editorSaveFeedback.path || '';
      const isEditorSave =
        (activePath && path && activePath === path) ||
        (this.editorSaveFeedback.state === 'saving' && (!feedbackPath || !path || feedbackPath === path)) ||
        (activePath && !path);
      if (!this.activeEditorTab || !isEditorSave) return;
      this.setEditorSaveFeedback('saved', path || activePath);
    },

    promptWriteFile() {
      const name = prompt('文件名');
      if (!name) return;
      const content = '';
      const target = this.joinRemotePath(this.activePath, name);
      this.sendSocket({ type: 'sftp:writefile', sessionId: this.activeSessionId, payload: { path: target, content } });
      setTimeout(() => this.refreshFiles(), 200);
    },

    promptMkdir() {
      const name = prompt('目录名');
      if (!name) return;
      const target = this.joinRemotePath(this.activePath, name);
      this.sendSocket({ type: 'sftp:mkdir', sessionId: this.activeSessionId, payload: { path: target } });
      setTimeout(() => this.refreshFiles(), 200);
    },

    renameEntry(entry) {
      const next = prompt('新名称', entry.filename);
      if (!next || next === entry.filename) return;
      const oldPath = this.joinRemotePath(this.activePath, entry.filename);
      const newPath = this.joinRemotePath(this.activePath, next);
      this.sendSocket({ type: 'sftp:rename', sessionId: this.activeSessionId, payload: { oldPath, newPath } });
      setTimeout(() => this.refreshFiles(), 200);
    },

    removeEntry(entry) {
      if (!confirm(`确认删除 ${entry.filename}？`)) return;
      const target = this.joinRemotePath(this.activePath, entry.filename);
      const type = entry.isDir ? 'sftp:rmdir' : 'sftp:unlink';
      this.sendSocket({ type, sessionId: this.activeSessionId, payload: { path: target } });
      setTimeout(() => this.refreshFiles(), 200);
    },

    showContextMenu(event, entry) {
      if (event) {
        event.preventDefault();
        event.stopPropagation();
      }
      const menuWidth = 175;
      const menuHeight = 260;
      const winW = window.innerWidth;
      const winH = window.innerHeight;

      let x = event?.clientX || 0;
      let y = event?.clientY || 0;

      const targetBtn = event?.target?.closest ? event.target.closest('button') : null;
      if (targetBtn) {
        const rect = targetBtn.getBoundingClientRect();
        if (rect.right + menuWidth > winW || rect.left + menuWidth > winW) {
          x = Math.max(12, rect.right - menuWidth);
        } else {
          x = Math.max(12, rect.left);
        }
        if (rect.bottom + menuHeight > winH - 12) {
          y = Math.max(12, rect.top - menuHeight);
        } else {
          y = rect.bottom + 4;
        }
      } else {
        if (x + menuWidth > winW - 12) {
          x = Math.max(12, winW - menuWidth - 12);
        }
        if (y + menuHeight > winH - 12) {
          y = Math.max(12, winH - menuHeight - 12);
        }
      }

      this.contextMenu.visible = true;
      this.contextMenu.x = Math.round(x);
      this.contextMenu.y = Math.round(y);
      this.contextMenu.entry = entry;
    },

    hideContextMenu() {
      this.contextMenu.visible = false;
      this.contextMenu.entry = null;
    },

    copyEntry(entry = this.contextMenu.entry) {
      if (!entry) return;
      const filePath = this.joinRemotePath(this.activePath, entry.filename);
      this.clipboard = { mode: 'copy', entries: [{ ...entry, path: filePath }] };
      this.hideContextMenu();
    },

    cutEntry(entry = this.contextMenu.entry) {
      if (!entry) return;
      const filePath = this.joinRemotePath(this.activePath, entry.filename);
      this.clipboard = { mode: 'cut', entries: [{ ...entry, path: filePath }] };
      this.hideContextMenu();
    },

    pasteClipboard() {
      if (!this.clipboard.mode || !this.clipboard.entries.length || !this.activeSessionId) return;
      const entry = this.clipboard.entries[0];
      const filename = entry.filename || entry.path.split('/').pop();
      const destination = this.joinRemotePath(this.activePath, filename);
      if (this.clipboard.mode === 'copy') {
        if (entry.isDir) {
          this.testMessage = '当前版本暂未支持目录复制，请复制单个文件。';
          this.testMessageType = 'error';
          return;
        }
        this.pendingPasteTarget = { sourcePath: entry.path, destinationPath: destination };
        this.sendSocket({ type: 'sftp:readfile', sessionId: this.activeSessionId, payload: { path: entry.path } });
      } else {
        this.sendSocket({ type: 'sftp:rename', sessionId: this.activeSessionId, payload: { oldPath: entry.path, newPath: destination } });
        this.clipboard = { mode: null, entries: [] };
      }
      this.hideContextMenu();
      setTimeout(() => this.refreshFiles(), 250);
    },

    downloadEntry(entry) {
      if (!this.activeSessionId) return;
      const filePath = this.joinRemotePath(this.activePath, entry.filename);
      const url = `/api/v1/files/download?sessionId=${encodeURIComponent(this.activeSessionId)}&path=${encodeURIComponent(filePath)}`;
      this.testMessage = `正在下载 ${entry.filename}...`;
      this.testMessageType = 'info';
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = entry.filename;
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
    },

    triggerUpload() {
      this.$refs.uploadInput?.click();
    },

    joinRemotePath(base, name) {
      const cleanName = String(name || '').replace(/^\/+/, '');
      if (!base || base === '.') return `./${cleanName}`;
      if (base === '/') return `/${cleanName}`;
      return `${String(base).replace(/\/$/, '')}/${cleanName}`;
    },

    async handleFileUpload(event) {
      const input = event.target;
      const file = input.files?.[0];
      if (!file || !this.activeSessionId) return;
      const form = new FormData();
      form.append('file', file);
      form.append('sessionId', this.activeSessionId);
      form.append('path', this.activePath || '.');
      this.uploading = true;
      this.uploadProgress = 15;
      this.testMessage = `正在上传 ${file.name}...`;
      this.testMessageType = 'info';
      try {
        const resp = await fetch('/api/v1/files/upload', { method: 'POST', body: form });
        const data = await resp.json();
        if (!resp.ok) throw new Error(data.message || '上传失败');
        this.uploadProgress = 100;
        this.testMessage = `上传完成 ${data.path}`;
        this.testMessageType = 'success';
        setTimeout(() => this.refreshFiles(), 200);
      } catch (error) {
        this.testMessage = error.message || '上传失败';
        this.testMessageType = 'error';
      } finally {
        this.uploading = false;
        input.value = '';
        setTimeout(() => { this.uploadProgress = 0; }, 400);
      }
    },

    refocusTerminal(force = false) {
      if (!this.activeSessionId) return;
      const id = this.activeSessionId;
      const term = this.terminals[id];
      if (!term) return;

      const focus = () => {
        try {
          term.focus();
          const helperTextarea = document.querySelector(`#terminal-${id} .xterm-helper-textarea`);
          if (helperTextarea) {
            helperTextarea.focus();
          }
        } catch (_) {}
      };

      if (force) focus();
      requestAnimationFrame(focus);
      setTimeout(focus, 10);
      setTimeout(focus, 50);
      setTimeout(focus, 150);
    },

    setupMobileBar(barEl) {
      if (!barEl || barEl._mobileBarInit) return;
      barEl._mobileBarInit = true;

      let touchStartX = 0;
      let touchStartY = 0;
      let scrollStartLeft = 0;
      let isDragging = false;
      let activeBtn = null;

      barEl.addEventListener('touchstart', (e) => {
        const btn = e.target.closest('button');
        if (!btn) return;
        // Crucial for iOS WebKit: calling preventDefault stops the virtual keyboard from dismissing/blurring!
        if (e.cancelable) e.preventDefault();
        e.stopPropagation();

        touchStartX = e.touches[0].clientX;
        touchStartY = e.touches[0].clientY;
        scrollStartLeft = barEl.scrollLeft;
        isDragging = false;
        activeBtn = btn;
        btn.classList.add('touch-active');
      }, { passive: false });

      barEl.addEventListener('touchmove', (e) => {
        if (!activeBtn) return;
        const touch = e.touches[0];
        const dx = touch.clientX - touchStartX;
        const dy = touch.clientY - touchStartY;

        if (Math.abs(dx) > 6 || Math.abs(dy) > 6) {
          isDragging = true;
          activeBtn.classList.remove('touch-active');
        }

        if (isDragging) {
          if (e.cancelable) e.preventDefault();
          barEl.scrollLeft = scrollStartLeft - dx;
        }
      }, { passive: false });

      const endTouch = (e) => {
        if (activeBtn) {
          activeBtn.classList.remove('touch-active');
          if (!isDragging) {
            const action = activeBtn.getAttribute('data-action');
            const payload = this.mobileKeyPayload(activeBtn);
            if (action) {
              this.handleMobileBarButton(action, payload, e);
            }
          }
          activeBtn = null;
        }
        isDragging = false;
      };

      barEl.addEventListener('touchend', endTouch, { passive: false });
      barEl.addEventListener('touchcancel', () => {
        if (activeBtn) activeBtn.classList.remove('touch-active');
        activeBtn = null;
        isDragging = false;
      }, { passive: true });

      // For desktop mouse interaction: prevent default to avoid stealing focus
      barEl.addEventListener('mousedown', (e) => {
        const btn = e.target.closest('button');
        if (btn && e.cancelable) {
          e.preventDefault();
        }
      });
    },

    hapticFeedback() {
      if (navigator.vibrate) {
        try { navigator.vibrate(10); } catch (_) {}
      }
    },

    mobileKeyPayload(btn) {
      if (!btn) return '';
      const key = btn.getAttribute('data-key');
      if (key) {
        const keyMap = {
          escape: '\x1b',
          tab: '\t',
          interrupt: '\x03',
          eof: '\x04',
          up: '\x1b[A',
          down: '\x1b[B',
          right: '\x1b[C',
          left: '\x1b[D',
          home: '\x1b[H',
          end: '\x1b[F',
          pageup: '\x1b[5~',
          pagedown: '\x1b[6~'
        };
        if (keyMap[key]) return keyMap[key];
      }
      return btn.getAttribute('data-payload') || '';
    },

    handleMobileBarClick(e) {
      const btn = e.target.closest('button');
      if (!btn) return;
      const action = btn.getAttribute('data-action');
      const payload = this.mobileKeyPayload(btn);
      if (action) {
        this.handleMobileBarButton(action, payload, e);
      }
    },

    handleMobileBarButton(action, payload = '', event = null) {
      if (event) {
        if (typeof event.preventDefault === 'function' && event.cancelable !== false) {
          event.preventDefault();
        }
        if (typeof event.stopPropagation === 'function') {
          event.stopPropagation();
        }
      }

      // De-duplicate multi-events (e.g. touchend followed by click)
      const now = Date.now();
      if (this._lastMobileBtnTime && (now - this._lastMobileBtnTime < 180)) {
        return;
      }
      this._lastMobileBtnTime = now;

      this.hapticFeedback();
      this.terminalMobileButton(action, payload);
    },

    toggleMobileKeyboard() {
      if (!this.activeSessionId) return;
      this.hapticFeedback();
      const id = this.activeSessionId;
      const helperTextarea = document.querySelector(`#terminal-${id} .xterm-helper-textarea`);
      if (this.isKeyboardOpen) {
        this.isKeyboardOpen = false;
        if (helperTextarea) {
          helperTextarea.blur();
        }
      } else {
        this.isKeyboardOpen = true;
        this.refocusTerminal(true);
      }
    },

    terminalMobileButton(action, payload = '') {
      if (action === 'keyboard') {
        this.toggleMobileKeyboard();
      } else if (action === 'files') {
        this.showMobileSftp = true;
        this.rightTab = 'files';
      } else if (action === 'copy') {
        this.openTerminalCopyViewer();
      } else if (action === 'paste') {
        this.mobileTerminalPaste();
      } else if (action === 'ctrl') {
        this.ctrlKeyActive = !this.ctrlKeyActive;
        if (this.ctrlKeyActive && !this.isKeyboardOpen) {
          this.refocusTerminal(true);
        }
      } else if (action === 'quick') {
        this.sendQuick(payload);
      }
    },

    async mobileTerminalPaste() {
      if (!this.activeSessionId) return;
      try {
        const text = await navigator.clipboard.readText();
        if (text) {
          this.pasteTextToSession(this.activeSessionId, text);
          return;
        }
        this.openTerminalPasteBox();
      } catch (err) {
        this.openTerminalPasteBox();
      }
    },

    async mobileTerminalCopy() {
      if (!this.activeSessionId || !this.terminals[this.activeSessionId]) return;
      this.openTerminalCopyViewer();
    },

    async sendQuick(command) {
      if (!this.activeSessionId) return;
      const noReturn = ['\x03', '\x04', '\x1b', '\x09', '\t', '\x1b[A', '\x1b[B', '\x1b[C', '\x1b[D', '\x1b[H', '\x1b[F', '\x1b[5~', '\x1b[6~'];
      if (noReturn.includes(command)) {
        await this.sendSocket({ type: 'ssh:input', sessionId: this.activeSessionId, payload: { data: command } });
      } else if (command === 'clear' || command === 'exit') {
        await this.sendSocket({ type: 'ssh:input', sessionId: this.activeSessionId, payload: { data: `${command}\r` } });
      } else {
        await this.sendSocket({ type: 'ssh:input', sessionId: this.activeSessionId, payload: { data: command } });
      }
    },

    percent(value) {
      return typeof value === 'number' ? `${Math.round(value)}%` : '-';
    },

    bytes(value) {
      if (typeof value !== 'number') return '-';
      if (value < 1024) return `${value.toFixed(0)} B/s`;
      if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB/s`;
      return `${(value / 1024 / 1024).toFixed(1)} MB/s`;
    },

    formatSize(value) {
      if (typeof value !== 'number') return '-';
      if (value < 1024) return `${value.toFixed(0)} B`;
      if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
      if (value < 1024 * 1024 * 1024) return `${(value / 1024 / 1024).toFixed(1)} MB`;
      return `${(value / 1024 / 1024 / 1024).toFixed(1)} GB`;
    },

    formatUnix(value) {
      if (!value) return '-';
      return new Date(value * 1000).toLocaleString();
    },

    getFileIcon(entry) {
      if (entry.isDir) return 'folder';
      const parts = (entry.filename || '').split('.');
      if (parts.length > 1) {
        const ext = parts.pop().toLowerCase();
        if (['jpg', 'jpeg', 'png', 'gif', 'svg', 'webp', 'ico'].includes(ext)) return 'image';
        if (['js', 'ts', 'jsx', 'tsx', 'go', 'py', 'c', 'cpp', 'rs', 'java', 'html', 'css', 'json', 'yaml', 'yml', 'sh', 'sql', 'md'].includes(ext)) return 'code';
        if (['zip', 'tar', 'gz', 'bz2', 'xz', '7z', 'rar'].includes(ext)) return 'archive';
        if (['mp3', 'wav', 'ogg', 'flac'].includes(ext)) return 'audio_file';
        if (['mp4', 'mkv', 'avi', 'mov'].includes(ext)) return 'video_file';
        if (['pdf'].includes(ext)) return 'picture_as_pdf';
      }
      return 'description';
    },

    getFileIconClass(entry) {
      return entry.isDir ? 'material-symbols-rounded file-icon folder' : 'material-symbols-rounded file-icon';
    }
  };
}
window.shadowApp = shadowApp;
