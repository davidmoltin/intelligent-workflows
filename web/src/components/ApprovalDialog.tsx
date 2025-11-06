import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { CheckCircle2, XCircle } from 'lucide-react'

interface ApprovalDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  action: 'approve' | 'reject'
  requestId: string
  onConfirm: (reason: string) => void
  isLoading?: boolean
}

export function ApprovalDialog({
  open,
  onOpenChange,
  action,
  requestId,
  onConfirm,
  isLoading = false,
}: ApprovalDialogProps) {
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')

  const handleConfirm = () => {
    // Validate reason
    if (!reason.trim()) {
      setError('Reason is required')
      return
    }

    if (reason.trim().length < 5) {
      setError('Reason must be at least 5 characters')
      return
    }

    onConfirm(reason.trim())
    handleClose()
  }

  const handleClose = () => {
    setReason('')
    setError('')
    onOpenChange(false)
  }

  const isApprove = action === 'approve'

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {isApprove ? (
              <CheckCircle2 className="h-5 w-5 text-green-600" />
            ) : (
              <XCircle className="h-5 w-5 text-red-600" />
            )}
            {isApprove ? 'Approve Request' : 'Reject Request'}
          </DialogTitle>
          <DialogDescription>
            {isApprove
              ? 'Please provide a reason for approving this request.'
              : 'Please provide a reason for rejecting this request.'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="request-id">Request ID</Label>
            <div className="font-mono text-xs text-muted-foreground rounded-md bg-muted p-2">
              {requestId}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="reason">
              Reason <span className="text-red-500">*</span>
            </Label>
            <Textarea
              id="reason"
              placeholder={
                isApprove
                  ? 'Enter your reason for approval...'
                  : 'Enter your reason for rejection...'
              }
              value={reason}
              onChange={(e) => {
                setReason(e.target.value)
                setError('')
              }}
              className={error ? 'border-red-500' : ''}
              rows={4}
              disabled={isLoading}
            />
            {error && (
              <p className="text-sm text-red-500">{error}</p>
            )}
            <p className="text-xs text-muted-foreground">
              Minimum 5 characters required
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={handleClose}
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button
            onClick={handleConfirm}
            disabled={isLoading}
            variant={isApprove ? 'default' : 'destructive'}
          >
            {isLoading ? (
              'Processing...'
            ) : isApprove ? (
              <>
                <CheckCircle2 className="mr-2 h-4 w-4" />
                Approve
              </>
            ) : (
              <>
                <XCircle className="mr-2 h-4 w-4" />
                Reject
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
