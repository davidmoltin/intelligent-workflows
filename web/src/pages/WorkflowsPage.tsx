import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useWorkflows, useDeleteWorkflow } from '@/api/hooks'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Plus, Trash2, Edit, Eye } from 'lucide-react'
import { DeleteWorkflowDialog } from '@/components/DeleteWorkflowDialog'

export function WorkflowsPage() {
  const { data: workflows, isLoading, error } = useWorkflows()
  const deleteWorkflow = useDeleteWorkflow()
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [workflowToDelete, setWorkflowToDelete] = useState<{ id: string; name: string } | null>(null)

  const handleDeleteClick = (id: string, name: string) => {
    setWorkflowToDelete({ id, name })
    setDeleteDialogOpen(true)
  }

  const handleDeleteConfirm = async () => {
    if (!workflowToDelete) return

    try {
      await deleteWorkflow.mutateAsync(workflowToDelete.id)
      setDeleteDialogOpen(false)
      setWorkflowToDelete(null)
    } catch (error) {
      console.error('Failed to delete workflow:', error)
    }
  }

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-muted-foreground">Loading workflows...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center">
        <Card className="border-destructive">
          <CardHeader>
            <CardTitle>Error Loading Workflows</CardTitle>
            <CardDescription>
              {error instanceof Error ? error.message : 'An error occurred'}
            </CardDescription>
          </CardHeader>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Workflows</h1>
          <p className="text-muted-foreground">
            Manage and monitor your intelligent workflows
          </p>
        </div>
        <Link to="/workflows/new">
          <Button>
            <Plus className="mr-2 h-4 w-4" />
            Create Workflow
          </Button>
        </Link>
      </div>

      {/* Stats */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Total Workflows</CardDescription>
            <CardTitle className="text-3xl">{workflows?.length || 0}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Active Workflows</CardDescription>
            <CardTitle className="text-3xl">
              {workflows?.filter((w) => w.enabled).length || 0}
            </CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Inactive Workflows</CardDescription>
            <CardTitle className="text-3xl">
              {workflows?.filter((w) => !w.enabled).length || 0}
            </CardTitle>
          </CardHeader>
        </Card>
      </div>

      {/* Workflows Table */}
      <Card>
        <CardHeader>
          <CardTitle>All Workflows</CardTitle>
          <CardDescription>
            A list of all workflows in your organization
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!workflows || workflows.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <p className="text-muted-foreground">No workflows found</p>
              <Link to="/workflows/new">
                <Button className="mt-4" variant="outline">
                  <Plus className="mr-2 h-4 w-4" />
                  Create your first workflow
                </Button>
              </Link>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Workflow ID</TableHead>
                  <TableHead>Version</TableHead>
                  <TableHead>Trigger</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {workflows.map((workflow) => (
                  <TableRow key={workflow.id}>
                    <TableCell className="font-medium">
                      {workflow.name}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {workflow.workflow_id}
                    </TableCell>
                    <TableCell>{workflow.version}</TableCell>
                    <TableCell>
                      <Badge variant="outline">
                        {workflow.definition.trigger.type}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant={workflow.enabled ? 'default' : 'secondary'}
                      >
                        {workflow.enabled ? 'Active' : 'Inactive'}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-2">
                        <Link to={`/workflows/${workflow.id}`}>
                          <Button variant="ghost" size="icon">
                            <Eye className="h-4 w-4" />
                          </Button>
                        </Link>
                        <Link to={`/workflows/${workflow.id}/edit`}>
                          <Button variant="ghost" size="icon">
                            <Edit className="h-4 w-4" />
                          </Button>
                        </Link>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleDeleteClick(workflow.id, workflow.name)}
                          disabled={deleteWorkflow.isPending}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Delete Dialog */}
      {workflowToDelete && (
        <DeleteWorkflowDialog
          open={deleteDialogOpen}
          onOpenChange={setDeleteDialogOpen}
          workflowName={workflowToDelete.name}
          onConfirm={handleDeleteConfirm}
          isDeleting={deleteWorkflow.isPending}
        />
      )}
    </div>
  )
}
