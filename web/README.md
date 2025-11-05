# Intelligent Workflows - Frontend

Modern React frontend for the Intelligent Workflows Service.

## Tech Stack

- **React 18** - UI library
- **TypeScript** - Type safety
- **Vite** - Build tool and dev server
- **Tailwind CSS** - Utility-first CSS framework
- **shadcn/ui** - Component library
- **React Router** - Client-side routing
- **TanStack Query (React Query)** - Data fetching and caching
- **React Flow** - Visual workflow builder (coming soon)
- **Recharts** - Analytics charts (coming soon)
- **Zustand** - State management
- **Lucide React** - Icons

## Getting Started

### Prerequisites

- Node.js 18+ and npm
- Backend API running on http://localhost:8080

### Installation

```bash
# Install dependencies
npm install
```

### Development

```bash
# Start development server (runs on http://localhost:3000)
npm run dev
```

The dev server includes:
- Hot Module Replacement (HMR)
- API proxy to backend (http://localhost:8080)

### Build

```bash
# Build for production
npm run build

# Preview production build
npm run preview
```

### Environment Variables

Create a `.env.local` file:

```env
# API Configuration
VITE_API_URL=http://localhost:8080/api/v1
```

## Project Structure

```
web/
├── src/
│   ├── api/              # API client and React Query hooks
│   │   ├── client.ts     # API client functions
│   │   └── hooks.ts      # React Query hooks
│   ├── components/       # React components
│   │   ├── layout/       # Layout components
│   │   └── ui/           # shadcn/ui components
│   ├── lib/              # Utility functions
│   │   └── utils.ts      # Class name merging, etc.
│   ├── pages/            # Page components
│   │   ├── WorkflowsPage.tsx
│   │   ├── ExecutionsPage.tsx
│   │   ├── ApprovalsPage.tsx
│   │   └── AnalyticsPage.tsx
│   ├── types/            # TypeScript type definitions
│   │   └── workflow.ts
│   ├── App.tsx           # Main app component with routing
│   ├── main.tsx          # App entry point
│   └── index.css         # Global styles
├── public/               # Static assets
├── components.json       # shadcn/ui configuration
├── tailwind.config.js    # Tailwind CSS configuration
├── tsconfig.json         # TypeScript configuration
└── vite.config.ts        # Vite configuration
```

## Features

### Workflows

- **List View**: Browse all workflows with status indicators
- **Statistics**: View total, active, and inactive workflows
- **Enable/Disable**: Toggle workflow activation
- **Delete**: Remove workflows
- **Create/Edit**: Visual workflow builder (coming soon)

### Executions

- **Dashboard**: Monitor workflow execution history
- **Statistics**: View completed, failed, and running executions
- **Filtering**: Filter by status, workflow, date range
- **Details**: View execution traces (coming soon)

### Approvals

- **Queue**: View pending approval requests
- **Actions**: Approve or reject with reasons
- **History**: View past approval decisions
- **Statistics**: Track approval metrics

### Analytics

- **Charts**: Workflow performance metrics (coming soon)
- **Reports**: Success rates, execution times, etc. (coming soon)

## API Client

The API client is built with TypeScript and provides type-safe access to all backend endpoints:

```typescript
import { workflowAPI } from '@/api/client'

// List workflows
const workflows = await workflowAPI.list()

// Get workflow by ID
const workflow = await workflowAPI.get(id)

// Create workflow
const newWorkflow = await workflowAPI.create(data)

// Update workflow
const updated = await workflowAPI.update(id, data)

// Delete workflow
await workflowAPI.delete(id)
```

## React Query Hooks

Use pre-configured hooks for data fetching:

```typescript
import { useWorkflows, useCreateWorkflow } from '@/api/hooks'

function MyComponent() {
  // Fetch workflows with automatic caching and refetching
  const { data, isLoading, error } = useWorkflows()

  // Create workflow mutation
  const createWorkflow = useCreateWorkflow()

  const handleCreate = async () => {
    await createWorkflow.mutateAsync(data)
  }
}
```

## UI Components

We use **shadcn/ui** components which are:
- Accessible (ARIA compliant)
- Customizable (Tailwind CSS)
- Copy-paste friendly

Example:

```typescript
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'

function Example() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>My Card</CardTitle>
      </CardHeader>
      <CardContent>
        <Button>Click me</Button>
      </CardContent>
    </Card>
  )
}
```

## Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build

## License

MIT
