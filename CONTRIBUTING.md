# Contributing to Open-Z3950-Gateway

Thank you for your interest in contributing! We welcome all contributions, from bug fixes to new features.

## Getting Started

1.  **Fork** the repository.
2.  **Clone** your fork locally.
3.  **Setup Environment**:
    *   Ideally, use **Docker Compose** (`docker compose up --build`) for a consistent dev environment.
    *   If developing locally, you need **Go 1.23+** and **Node.js 18+**.

## Development Workflow

1.  **Frontend Changes**:
    *   Work in the `webapp/` directory.
    *   Run `npm run dev` for hot-reloading (you may need to configure proxy in `vite.config.ts` to point to a running backend).
2.  **Backend Changes**:
    *   Work in `cmd/` or `pkg/`.
    *   After changes, run `go build ./cmd/gateway` to verify.
3.  **Commit Messages**:
    *   We follow [Conventional Commits](https://www.conventionalcommits.org/).
    *   Example: `feat: add support for UNIMARC encoding` or `fix: resolve null pointer in search handler`.

## Pull Requests

1.  Create a new branch: `git checkout -b feat/my-feature`.
2.  Push to your fork.
3.  Open a Pull Request against the `master` branch.
4.  Ensure CI checks pass.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
