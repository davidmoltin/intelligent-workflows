import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { BarChart3 } from 'lucide-react'

export function AnalyticsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Analytics</h1>
        <p className="text-muted-foreground">
          Insights and metrics for your workflows
        </p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5 text-muted-foreground" />
            <CardTitle>Analytics Dashboard</CardTitle>
          </div>
          <CardDescription>
            Coming soon: Comprehensive analytics and reporting
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <BarChart3 className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground">Analytics features coming soon</p>
            <p className="mt-2 text-sm text-muted-foreground">
              We're working on bringing you detailed insights about your workflow performance
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
