import { useApprovals, useApproveRequest, useRejectRequest } from '@/api/hooks'
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
import { ApprovalDialog } from '@/components/ApprovalDialog'
import { CheckCircle2, XCircle } from 'lucide-react'
import { useState } from 'react'

type DialogState = {
  open: boolean
  action: 'approve' | 'reject'
  approvalId: string
  requestId: string
}

export function ApprovalsPage() {
  const { data: approvals, isLoading, error } = useApprovals()
  const approveRequest = useApproveRequest()
  const rejectRequest = useRejectRequest()
  const [processingId, setProcessingId] = useState<string | null>(null)
  const [dialogState, setDialogState] = useState<DialogState>({
    open: false,
    action: 'approve',
    approvalId: '',
    requestId: '',
  })

  const handleApprove = (id: string, requestId: string) => {
    setDialogState({
      open: true,
      action: 'approve',
      approvalId: id,
      requestId,
    })
  }

  const handleReject = (id: string, requestId: string) => {
    setDialogState({
      open: true,
      action: 'reject',
      approvalId: id,
      requestId,
    })
  }

  const handleConfirm = async (reason: string) => {
    const { approvalId, action } = dialogState
    setProcessingId(approvalId)

    try {
      if (action === 'approve') {
        await approveRequest.mutateAsync({ id: approvalId, reason })
      } else {
        await rejectRequest.mutateAsync({ id: approvalId, reason })
      }
    } catch (error) {
      console.error(`Failed to ${action}:`, error)
    } finally {
      setProcessingId(null)
    }
  }

  const getStatusVariant = (status: string) => {
    switch (status) {
      case 'approved':
        return 'default'
      case 'rejected':
        return 'destructive'
      case 'pending':
        return 'secondary'
      case 'expired':
        return 'outline'
      default:
        return 'outline'
    }
  }

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-muted-foreground">Loading approvals...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center">
        <Card className="border-destructive">
          <CardHeader>
            <CardTitle>Error Loading Approvals</CardTitle>
            <CardDescription>
              {error instanceof Error ? error.message : 'An error occurred'}
            </CardDescription>
          </CardHeader>
        </Card>
      </div>
    )
  }

  const pendingApprovals = approvals?.filter((a) => a.status === 'pending') || []
  const completedApprovals = approvals?.filter((a) => a.status !== 'pending') || []

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Approvals</h1>
        <p className="text-muted-foreground">
          Review and approve workflow execution requests
        </p>
      </div>

      {/* Stats */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Pending</CardDescription>
            <CardTitle className="text-3xl">{pendingApprovals.length}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Approved</CardDescription>
            <CardTitle className="text-3xl">
              {approvals?.filter((a) => a.status === 'approved').length || 0}
            </CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Rejected</CardDescription>
            <CardTitle className="text-3xl">
              {approvals?.filter((a) => a.status === 'rejected').length || 0}
            </CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Expired</CardDescription>
            <CardTitle className="text-3xl">
              {approvals?.filter((a) => a.status === 'expired').length || 0}
            </CardTitle>
          </CardHeader>
        </Card>
      </div>

      {/* Pending Approvals */}
      {pendingApprovals.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Pending Approvals</CardTitle>
            <CardDescription>
              Requests awaiting your review and decision
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Request ID</TableHead>
                  <TableHead>Entity</TableHead>
                  <TableHead>Reason</TableHead>
                  <TableHead>Requested At</TableHead>
                  <TableHead>Expires At</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {pendingApprovals.map((approval) => (
                  <TableRow key={approval.id}>
                    <TableCell className="font-mono text-xs">
                      {approval.request_id}
                    </TableCell>
                    <TableCell>
                      <div className="text-sm">
                        <div className="font-medium">{approval.entity_type}</div>
                        <div className="text-muted-foreground">
                          {approval.entity_id}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="max-w-xs truncate">
                      {approval.reason}
                    </TableCell>
                    <TableCell>
                      {new Date(approval.requested_at).toLocaleString()}
                    </TableCell>
                    <TableCell>
                      {approval.expires_at
                        ? new Date(approval.expires_at).toLocaleString()
                        : '-'}
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-2">
                        <Button
                          size="sm"
                          onClick={() => handleApprove(approval.id, approval.request_id)}
                          disabled={
                            processingId === approval.id ||
                            approveRequest.isPending
                          }
                        >
                          <CheckCircle2 className="mr-1 h-4 w-4" />
                          Approve
                        </Button>
                        <Button
                          size="sm"
                          variant="destructive"
                          onClick={() => handleReject(approval.id, approval.request_id)}
                          disabled={
                            processingId === approval.id ||
                            rejectRequest.isPending
                          }
                        >
                          <XCircle className="mr-1 h-4 w-4" />
                          Reject
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Completed Approvals */}
      <Card>
        <CardHeader>
          <CardTitle>Approval History</CardTitle>
          <CardDescription>
            Previously reviewed approval requests
          </CardDescription>
        </CardHeader>
        <CardContent>
          {completedApprovals.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <p className="text-muted-foreground">No completed approvals</p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Request ID</TableHead>
                  <TableHead>Entity</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Decision Reason</TableHead>
                  <TableHead>Decided At</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {completedApprovals.map((approval) => (
                  <TableRow key={approval.id}>
                    <TableCell className="font-mono text-xs">
                      {approval.request_id}
                    </TableCell>
                    <TableCell>
                      <div className="text-sm">
                        <div className="font-medium">{approval.entity_type}</div>
                        <div className="text-muted-foreground">
                          {approval.entity_id}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={getStatusVariant(approval.status)}>
                        {approval.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="max-w-xs truncate">
                      {approval.decision_reason || '-'}
                    </TableCell>
                    <TableCell>
                      {approval.decided_at
                        ? new Date(approval.decided_at).toLocaleString()
                        : '-'}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Approval Dialog */}
      <ApprovalDialog
        open={dialogState.open}
        onOpenChange={(open) => setDialogState({ ...dialogState, open })}
        action={dialogState.action}
        requestId={dialogState.requestId}
        onConfirm={handleConfirm}
        isLoading={processingId === dialogState.approvalId}
      />
    </div>
  )
}
