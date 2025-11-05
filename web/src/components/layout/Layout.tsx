import { Link, useLocation } from 'react-router-dom'
import { cn } from '@/lib/utils'
import {
  Workflow,
  BarChart3,
  CheckCircle2,
  Activity,
  Settings,
  FileText,
} from 'lucide-react'

const navigation = [
  { name: 'Workflows', href: '/workflows', icon: Workflow },
  { name: 'Executions', href: '/executions', icon: Activity },
  { name: 'Approvals', href: '/approvals', icon: CheckCircle2 },
  { name: 'Analytics', href: '/analytics', icon: BarChart3 },
  { name: 'Documentation', href: '/docs', icon: FileText },
  { name: 'Settings', href: '/settings', icon: Settings },
]

export function Layout({ children }: { children: React.ReactNode }) {
  const location = useLocation()

  return (
    <div className="flex h-screen bg-background">
      {/* Sidebar */}
      <div className="w-64 border-r bg-card">
        <div className="flex h-full flex-col">
          {/* Logo */}
          <div className="flex h-16 items-center border-b px-6">
            <div className="flex items-center gap-2">
              <Workflow className="h-6 w-6 text-primary" />
              <span className="text-lg font-semibold">Workflows</span>
            </div>
          </div>

          {/* Navigation */}
          <nav className="flex-1 space-y-1 p-4">
            {navigation.map((item) => {
              const Icon = item.icon
              const isActive = location.pathname.startsWith(item.href)

              return (
                <Link
                  key={item.name}
                  to={item.href}
                  className={cn(
                    'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-primary text-primary-foreground'
                      : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                  )}
                >
                  <Icon className="h-5 w-5" />
                  {item.name}
                </Link>
              )
            })}
          </nav>

          {/* Footer */}
          <div className="border-t p-4">
            <div className="text-xs text-muted-foreground">
              <div>Version 0.1.0</div>
              <div className="mt-1">
                <a
                  href="https://github.com/yourusername/intelligent-workflows"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="hover:text-foreground"
                >
                  GitHub
                </a>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        <main className="flex-1 overflow-y-auto">
          <div className="container mx-auto p-6">{children}</div>
        </main>
      </div>
    </div>
  )
}
