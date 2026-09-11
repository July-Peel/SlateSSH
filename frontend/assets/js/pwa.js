// The worker only stores the public application shell. SSH and API traffic stay online.
window.slatePwa = {
  registration: null,
  installPrompt: null,
  async register() {
    if (!('serviceWorker' in navigator) || !window.isSecureContext) return;
    try {
      const hadController = !!navigator.serviceWorker.controller;
      const registration = await navigator.serviceWorker.register('/sw.js', { updateViaCache: 'none' });
      this.registration = registration;
      const announceUpdate = () => {
        if (registration.waiting && hadController) {
          window.dispatchEvent(new Event('slate:update-ready'));
        }
      };
      announceUpdate();
      registration.addEventListener('updatefound', () => {
        const worker = registration.installing;
        worker?.addEventListener('statechange', announceUpdate);
      });
      document.addEventListener('visibilitychange', () => {
        if (!document.hidden && navigator.onLine) registration.update().catch(() => {});
      });
    } catch (error) {
      console.warn('SlateSSH offline shell unavailable:', error);
    }
  },
  applyUpdate() {
    const worker = this.registration?.waiting;
    if (!worker) return;
    navigator.serviceWorker.addEventListener('controllerchange', () => location.reload(), { once: true });
    worker.postMessage({ type: 'SKIP_WAITING' });
  }
};
window.addEventListener('beforeinstallprompt', event => {
  event.preventDefault();
  window.slatePwa.installPrompt = event;
});
window.addEventListener('load', () => window.slatePwa.register(), { once: true });
