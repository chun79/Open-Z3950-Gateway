# Open-Z3950-Gateway Frontend

This directory contains the React-based frontend application for the Open-Z3950-Gateway. It is built using **Vite**, **React**, and **TypeScript**.

## 🚀 Getting Started

### Prerequisites

*   **Node.js**: v18 or higher
*   **npm**: v9 or higher

### Installation

Navigate to the `webapp` directory and install dependencies:

```bash
cd webapp
npm install
```

## 🛠 Development

To start the development server with hot module replacement (HMR):

```bash
npm run dev
```

The application will generally be available at `http://localhost:5173`.

> **Note**: For the frontend to communicate with the backend API during development, ensure your backend service is running (usually on port `8899`). You may need to configure the proxy settings in `vite.config.ts` if CORS issues arise, or ensure the backend is serving the frontend correctly in production.

## 📦 Building for Production

To build the application for production deployment:

```bash
npm run build
```

The output files will be generated in the `dist/` directory. These static files are intended to be embedded into the Go backend binary or served via a standard web server (Nginx, Apache).

## 📂 Project Structure

*   `src/components`: Reusable UI components.
*   `src/context`: React Context definitions (Auth, I18n).
*   `src/locales`: Internationalization JSON files (en, zh).
*   `src/pages`: Main application views (Search, Browse, Login, etc.).
*   `src/utils`: Helper functions and utilities.
*   `src/App.tsx`: Main application component and routing logic.
*   `src/main.tsx`: Entry point.

## 🔧 Scripts

*   `npm run dev`: Start development server.
*   `npm run build`: Build for production.
*   `npm run preview`: Preview the production build locally.
*   `npm run lint`: Run ESLint to check for code quality issues.
