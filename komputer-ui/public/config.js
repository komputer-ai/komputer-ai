// Runtime configuration — mounted as a ConfigMap in k8s.
// Override these values for your environment.
window.__KOMPUTER_CONFIG__ = {
  apiUrl: "http://localhost:8080",  // Override via ConfigMap in k8s
};
