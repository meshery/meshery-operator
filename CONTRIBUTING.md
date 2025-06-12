# 🌟 Contributing to Meshery Operator

Welcome! 👋 We're thrilled that you're considering contributing to the **Meshery Operator**, a component of the [Meshery](https://meshery.io) project under the CNCF umbrella.

This guide will walk you through setting up the Meshery Operator, understanding the codebase, and making successful contributions. Whether you're fixing bugs, adding features, or improving docs — we’re happy to have you!

---

## 📖 About Meshery Operator

The **Meshery Operator** acts as a controller that manages the lifecycle of service meshes. It provides dynamic configuration and orchestration support to Meshery and the service mesh components it manages.

It is implemented in Go and follows typical Kubernetes operator patterns.

---

## 🧰 Prerequisites

Make sure the following tools are installed on your system:

- [Go (>= 1.20)](https://go.dev/doc/install)
- [Docker](https://docs.docker.com/get-docker/)
- [GNU Make](https://www.gnu.org/software/make/)
- [Git](https://git-scm.com/)
- Kubernetes (local or remote cluster like [kind](https://kind.sigs.k8s.io/), [minikube](https://minikube.sigs.k8s.io/), or GKE)

---

## 📂 Project Structure (Important Folders)

| Folder/File         | Purpose                                               |
|---------------------|-------------------------------------------------------|
| `main.go`           | Entry point of the operator                           |
| `config/`           | Helm chart and manifests                              |
| `pkg/meshery`       | Core logic for handling service mesh deployments      |
| `controllers/`      | Operator controller logic                             |
| `test/`             | Unit and integration test cases                       |
| `Makefile`          | Common build/test/dev commands                        |

---

## 🚀 Local Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/meshery/meshery-operator.git
   cd meshery-operator

## 🛠️ Building and Running the Operator

Once inside the project directory, follow these steps:

### 🔨 Build the project

This will compile the operator and prepare it for local development.

```bash
make build
