import { Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from './components/layout/Layout'
import { WorkflowsPage } from './pages/WorkflowsPage'
import { CreateWorkflowPage } from './pages/CreateWorkflowPage'
import { WorkflowDetailPage } from './pages/WorkflowDetailPage'
import { EditWorkflowPage } from './pages/EditWorkflowPage'
import { ExecutionsPage } from './pages/ExecutionsPage'
import { ExecutionDetailPage } from './pages/ExecutionDetailPage'
import { ApprovalsPage } from './pages/ApprovalsPage'
import { AnalyticsPage } from './pages/AnalyticsPage'

function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<Navigate to="/workflows" replace />} />
        <Route path="/workflows" element={<WorkflowsPage />} />
        <Route path="/workflows/new" element={<CreateWorkflowPage />} />
        <Route path="/workflows/:id" element={<WorkflowDetailPage />} />
        <Route path="/workflows/:id/edit" element={<EditWorkflowPage />} />
        <Route path="/executions" element={<ExecutionsPage />} />
        <Route path="/executions/:id" element={<ExecutionDetailPage />} />
        <Route path="/approvals" element={<ApprovalsPage />} />
        <Route path="/analytics" element={<AnalyticsPage />} />
        <Route path="*" element={<NotFound />} />
      </Routes>
    </Layout>
  )
}

function NotFound() {
  return (
    <div className="flex h-full items-center justify-center">
      <div className="text-center">
        <h1 className="text-4xl font-bold">404</h1>
        <p className="mt-2 text-muted-foreground">Page not found</p>
      </div>
    </div>
  )
}

export default App
