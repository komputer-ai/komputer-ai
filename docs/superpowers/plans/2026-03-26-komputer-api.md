# komputer-api Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Gin-based Go API that creates/queries KomputerAgent CRs in Kubernetes and consumes agent events from a Redis queue.

**Architecture:** Simple flat Go project with a Gin HTTP server and a Redis consumer goroutine. The API talks to Kubernetes to create/list KomputerAgent CRs, and can forward tasks to existing agent pods. The Redis worker logs events for now, designed to be extracted into komputer-event-handler later.

**Tech Stack:** Go, Gin, client-go, go-redis, CRD types from komputer-operator

---

## File Structure

All files live under `komputer-api/` in the monorepo. Flat structure, no nested packages.

| File | Responsibility |
|------|---------------|
| `main.go` | Entrypoint: starts HTTP server + Redis worker goroutine |
| `handler.go` | Gin route handlers for POST and GET /api/v1/agents |
| `k8s.go` | Kubernetes client: create, list, get KomputerAgent CRs, get pod IP |
| `worker.go` | Redis consumer goroutine — reads from queue, logs messages |
| `go.mod` | Go module definition |
| `Dockerfile` | Container image build |

---

### Task 1: Initialize the Go module and dependencies

**Files:**
- Create: `komputer-api/go.mod`
- Create: `komputer-api/main.go` (skeleton)

- [ ] **Step 1: Create the komputer-api directory and initialize Go module**

```bash
cd /Users/amitdebachar/Documents/projects/komputer-ai
mkdir -p komputer-api
cd komputer-api
go mod init github.com/komputer-ai/komputer-api
```

- [ ] **Step 2: Add dependencies**

```bash
cd /Users/amitdebachar/Documents/projects/komputer-ai/komputer-api
go get github.com/gin-gonic/gin
go get github.com/redis/go-redis/v9
go get k8s.io/client-go@latest
go get k8s.io/apimachinery@latest
go get sigs.k8s.io/controller-runtime
```

We also need the operator's CRD types. Since the monorepo has strict isolation (no shared code), we'll use a `replace` directive pointing to the local operator module:

```bash
cd /Users/amitdebachar/Documents/projects/komputer-ai/komputer-api
go mod edit -require github.com/komputer-ai/komputer-operator@v0.0.0
go mod edit -replace github.com/komputer-ai/komputer-operator=../komputer-operator
go mod tidy
```

- [ ] **Step 3: Create skeleton main.go**

Create `komputer-api/main.go`:

```go
package main

import (
	"log"
	"os"
)

func main() {
	log.Println("komputer-api starting...")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("listening on :%s", port)
}
```

- [ ] **Step 4: Verify it compiles**

```bash
cd /Users/amitdebachar/Documents/projects/komputer-ai/komputer-api
go build ./...
```

- [ ] **Step 5: Commit**

```bash
cd /Users/amitdebachar/Documents/projects/komputer-ai
git add komputer-api/
git commit -m "feat(api): initialize komputer-api Go module with dependencies"
```

---

### Task 2: Implement the Kubernetes client

**Files:**
- Create: `komputer-api/k8s.go`

- [ ] **Step 1: Create k8s.go with client initialization and CRUD operations**

Create `komputer-api/k8s.go`:

```go
package main

import (
	"context"
	"fmt"
	"net/http"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// K8sClient wraps a controller-runtime client for KomputerAgent operations.
type K8sClient struct {
	client    client.Client
	namespace string
}

// NewK8sClient creates a new Kubernetes client configured for KomputerAgent CRs.
func NewK8sClient(namespace string) (*K8sClient, error) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(komputerv1alpha1.AddToScheme(scheme))

	config, err := ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	c, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	return &K8sClient{client: c, namespace: namespace}, nil
}

// CreateAgent creates a new KomputerAgent CR.
func (k *K8sClient) CreateAgent(ctx context.Context, name, instructions, model, templateRef string) (*komputerv1alpha1.KomputerAgent, error) {
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	if templateRef == "" {
		templateRef = "default"
	}

	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: k.namespace,
			Labels: map[string]string{
				"komputer.ai/agent-name": name,
			},
		},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			TemplateRef:  templateRef,
			Instructions: instructions,
			Model:        model,
		},
	}

	if err := k.client.Create(ctx, agent); err != nil {
		return nil, err
	}
	return agent, nil
}

// GetAgent gets a KomputerAgent by name.
func (k *K8sClient) GetAgent(ctx context.Context, name string) (*komputerv1alpha1.KomputerAgent, error) {
	agent := &komputerv1alpha1.KomputerAgent{}
	err := k.client.Get(ctx, types.NamespacedName{Name: name, Namespace: k.namespace}, agent)
	if err != nil {
		return nil, err
	}
	return agent, nil
}

// ListAgents returns all KomputerAgent CRs in the namespace.
func (k *K8sClient) ListAgents(ctx context.Context) ([]komputerv1alpha1.KomputerAgent, error) {
	list := &komputerv1alpha1.KomputerAgentList{}
	if err := k.client.List(ctx, list, client.InNamespace(k.namespace)); err != nil {
		return nil, err
	}
	return list.Items, nil
}

// GetAgentPodIP returns the IP of the pod running the given agent.
func (k *K8sClient) GetAgentPodIP(ctx context.Context, podName string) (string, error) {
	pod := &corev1.Pod{}
	err := k.client.Get(ctx, types.NamespacedName{Name: podName, Namespace: k.namespace}, pod)
	if err != nil {
		return "", err
	}
	if pod.Status.PodIP == "" {
		return "", fmt.Errorf("pod %s has no IP yet", podName)
	}
	return pod.Status.PodIP, nil
}

// ForwardTaskToAgent sends a task to an existing agent's FastAPI endpoint.
func (k *K8sClient) ForwardTaskToAgent(ctx context.Context, podIP, instructions string) error {
	url := fmt.Sprintf("http://%s:8000/task", podIP)
	body := fmt.Sprintf(`{"instructions":%q}`, instructions)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to forward task to agent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("agent returned status %d", resp.StatusCode)
	}
	return nil
}
```

Note: Add `"strings"` to the imports.

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/amitdebachar/Documents/projects/komputer-ai/komputer-api
go build ./...
```

- [ ] **Step 3: Commit**

```bash
cd /Users/amitdebachar/Documents/projects/komputer-ai
git add komputer-api/
git commit -m "feat(api): implement Kubernetes client for KomputerAgent CRs"
```

---

### Task 3: Implement the Gin route handlers

**Files:**
- Create: `komputer-api/handler.go`

- [ ] **Step 1: Create handler.go with POST and GET routes**

Create `komputer-api/handler.go`:

```go
package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
)

// CreateAgentRequest is the request body for POST /api/v1/agents.
type CreateAgentRequest struct {
	Name         string `json:"name" binding:"required"`
	Instructions string `json:"instructions" binding:"required"`
	Model        string `json:"model"`
	TemplateRef  string `json:"templateRef"`
}

// AgentResponse is the response for agent endpoints.
type AgentResponse struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Model     string `json:"model"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

// AgentListResponse is the response for GET /api/v1/agents.
type AgentListResponse struct {
	Agents []AgentResponse `json:"agents"`
}

// SetupRoutes registers all API routes on the given Gin engine.
func SetupRoutes(r *gin.Engine, k8s *K8sClient) {
	v1 := r.Group("/api/v1")
	{
		v1.POST("/agents", createOrTriggerAgent(k8s))
		v1.GET("/agents", listAgents(k8s))
	}
}

func createOrTriggerAgent(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateAgentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if agent already exists
		existing, err := k8s.GetAgent(c.Request.Context(), req.Name)
		if err != nil && !errors.IsNotFound(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check agent: " + err.Error()})
			return
		}

		if existing != nil {
			// Agent exists — forward task to the running pod
			if existing.Status.PodName == "" {
				c.JSON(http.StatusConflict, gin.H{"error": "agent exists but has no running pod yet"})
				return
			}

			podIP, err := k8s.GetAgentPodIP(c.Request.Context(), existing.Status.PodName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get agent pod IP: " + err.Error()})
				return
			}

			if err := k8s.ForwardTaskToAgent(c.Request.Context(), podIP, req.Instructions); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to forward task: " + err.Error()})
				return
			}

			log.Printf("forwarded task to existing agent %s", req.Name)
			c.JSON(http.StatusOK, AgentResponse{
				Name:      existing.Name,
				Namespace: existing.Namespace,
				Model:     existing.Spec.Model,
				Status:    string(existing.Status.Phase),
				CreatedAt: existing.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
			})
			return
		}

		// Agent doesn't exist — create new
		agent, err := k8s.CreateAgent(c.Request.Context(), req.Name, req.Instructions, req.Model, req.TemplateRef)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create agent: " + err.Error()})
			return
		}

		log.Printf("created new agent %s", req.Name)
		c.JSON(http.StatusCreated, AgentResponse{
			Name:      agent.Name,
			Namespace: agent.Namespace,
			Model:     agent.Spec.Model,
			Status:    "Pending",
			CreatedAt: agent.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
		})
	}
}

func listAgents(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		agents, err := k8s.ListAgents(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list agents: " + err.Error()})
			return
		}

		resp := AgentListResponse{Agents: make([]AgentResponse, 0, len(agents))}
		for _, a := range agents {
			resp.Agents = append(resp.Agents, AgentResponse{
				Name:      a.Name,
				Namespace: a.Namespace,
				Model:     a.Spec.Model,
				Status:    string(a.Status.Phase),
				CreatedAt: a.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
			})
		}

		c.JSON(http.StatusOK, resp)
	}
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/amitdebachar/Documents/projects/komputer-ai/komputer-api
go build ./...
```

- [ ] **Step 3: Commit**

```bash
cd /Users/amitdebachar/Documents/projects/komputer-ai
git add komputer-api/
git commit -m "feat(api): implement Gin route handlers for agent CRUD"
```

---

### Task 4: Implement the Redis event worker

**Files:**
- Create: `komputer-api/worker.go`

- [ ] **Step 1: Create worker.go with Redis consumer**

Create `komputer-api/worker.go`:

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisWorkerConfig holds the configuration for the Redis event worker.
type RedisWorkerConfig struct {
	Address  string
	Password string
	DB       int
	Queue    string
}

// StartRedisWorker starts a goroutine that consumes messages from the Redis queue and logs them.
// It is designed to be easily extracted into a separate komputer-event-handler service later.
func StartRedisWorker(ctx context.Context, cfg RedisWorkerConfig) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	go func() {
		log.Printf("redis worker started, consuming from queue %q at %s", cfg.Queue, cfg.Address)

		for {
			select {
			case <-ctx.Done():
				log.Println("redis worker shutting down")
				rdb.Close()
				return
			default:
			}

			// BLPop blocks until a message is available or timeout
			result, err := rdb.BLPop(ctx, 5*time.Second, cfg.Queue).Result()
			if err != nil {
				if err == redis.Nil || err == context.Canceled {
					continue
				}
				log.Printf("redis worker error: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// result[0] is the queue name, result[1] is the message
			if len(result) >= 2 {
				log.Printf("agent event: %s", result[1])
			}
		}
	}()
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/amitdebachar/Documents/projects/komputer-ai/komputer-api
go build ./...
```

- [ ] **Step 3: Commit**

```bash
cd /Users/amitdebachar/Documents/projects/komputer-ai
git add komputer-api/
git commit -m "feat(api): implement Redis event worker (logs messages)"
```

---

### Task 5: Wire everything together in main.go

**Files:**
- Modify: `komputer-api/main.go`

- [ ] **Step 1: Update main.go to initialize all components and start the server**

Replace `komputer-api/main.go`:

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("komputer-api starting...")

	// Configuration from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}
	redisAddr := os.Getenv("REDIS_ADDRESS")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisQueue := os.Getenv("REDIS_QUEUE")
	if redisQueue == "" {
		redisQueue = "komputer-events"
	}

	// Context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Kubernetes client
	k8s, err := NewK8sClient(namespace)
	if err != nil {
		log.Fatalf("failed to create k8s client: %v", err)
	}
	log.Println("kubernetes client initialized")

	// Start Redis event worker
	StartRedisWorker(ctx, RedisWorkerConfig{
		Address:  redisAddr,
		Password: redisPassword,
		DB:       0,
		Queue:    redisQueue,
	})
	log.Println("redis worker started")

	// Setup Gin router
	r := gin.Default()
	SetupRoutes(r, k8s)

	// Graceful shutdown on SIGINT/SIGTERM
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		cancel()
	}()

	log.Printf("listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

- [ ] **Step 2: Run go mod tidy and verify it compiles**

```bash
cd /Users/amitdebachar/Documents/projects/komputer-ai/komputer-api
go mod tidy
go build ./...
```

- [ ] **Step 3: Commit**

```bash
cd /Users/amitdebachar/Documents/projects/komputer-ai
git add komputer-api/
git commit -m "feat(api): wire up main.go with Gin server, k8s client, and Redis worker"
```

---

### Task 6: Add Dockerfile

**Files:**
- Create: `komputer-api/Dockerfile`

- [ ] **Step 1: Create multi-stage Dockerfile**

Create `komputer-api/Dockerfile`:

```dockerfile
FROM golang:1.22 AS builder

WORKDIR /app

# Copy the operator module first (for the CRD types)
COPY komputer-operator/ /komputer-operator/

# Copy and build the API
COPY komputer-api/ .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o komputer-api .

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /app/komputer-api .
USER 65532:65532
ENTRYPOINT ["/komputer-api"]
```

Note: This Dockerfile is built from the monorepo root with context `.`, e.g.:
`docker build -f komputer-api/Dockerfile .`

- [ ] **Step 2: Verify the build still works**

```bash
cd /Users/amitdebachar/Documents/projects/komputer-ai/komputer-api
go build ./...
```

- [ ] **Step 3: Commit**

```bash
cd /Users/amitdebachar/Documents/projects/komputer-ai
git add komputer-api/
git commit -m "feat(api): add Dockerfile for komputer-api"
```

---

## Verification

After completing all tasks:

1. **Build succeeds:**
   ```bash
   cd komputer-api && go build ./...
   ```

2. **Binary runs (will fail connecting to k8s/redis but should start):**
   ```bash
   ./komputer-api
   ```
   Expected: Logs "komputer-api starting..." then fails with kubeconfig error (expected outside cluster).

3. **Files match structure:**
   ```
   komputer-api/
   ├── main.go
   ├── handler.go
   ├── k8s.go
   ├── worker.go
   ├── go.mod
   ├── go.sum
   └── Dockerfile
   ```
