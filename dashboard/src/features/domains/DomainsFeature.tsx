import { useMemo, useState } from "react"
import {
  AlertCircle,
  CheckCircle2,
  Clock,
  RefreshCcw,
  Search,
  ShieldCheck,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Card, CardContent } from "@/components/ui/card"
import { useDomains } from "@/hooks/useDomains"
import { cn } from "@/lib/utils"

function StatusBadge({ status }: { status: string }) {
  const s = status.toLowerCase()
  if (s === "completed" || s === "verified") {
    return (
      <Badge className="border-vn-success/30 bg-vn-success-soft text-vn-success">
        <CheckCircle2 className="mr-1 size-3" />
        Verified
      </Badge>
    )
  }
  if (s === "pending" || s === "ip_applied") {
    return (
      <Badge className="border-vn-warn/30 bg-vn-warn-soft text-vn-warn">
        <Clock className="mr-1 size-3" />
        {s === "ip_applied" ? "Awaiting verify" : "Pending"}
      </Badge>
    )
  }
  if (s === "failed") {
    return (
      <Badge className="border-vn-danger/30 bg-vn-danger-soft text-vn-danger">
        <AlertCircle className="mr-1 size-3" />
        Failed
      </Badge>
    )
  }
  return <Badge variant="outline">{status}</Badge>
}

export function DomainsFeature() {
  const { domains, isLoading, error, refresh, verify } = useDomains()
  const [query, setQuery] = useState("")
  const [verifying, setVerifying] = useState<string | null>(null)
  const [flash, setFlash] = useState<string | null>(null)

  const filtered = useMemo(
    () =>
      domains.filter((d) =>
        d.domain_name.toLowerCase().includes(query.toLowerCase()),
      ),
    [domains, query],
  )

  const handleVerify = async (domain: string) => {
    setVerifying(domain)
    setFlash(null)
    try {
      const res = await verify(domain)
      setFlash(res.message ?? `Verification triggered for ${domain}.`)
    } catch (err) {
      setFlash(err instanceof Error ? err.message : "Verification failed")
    } finally {
      setVerifying(null)
    }
  }

  return (
    <div>
      <section className="mb-5 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <span className="vn-eyebrow mb-2">domains</span>
          <h1 className="mt-2 text-3xl font-[540] tracking-[-0.035em] text-vn-text md:text-4xl">
            Domains
          </h1>
          <p className="mt-2 max-w-2xl text-vn-text-2">
            Manage custom domains and DNS verification status. Domains are
            added by your customers through the connect API.
          </p>
        </div>

        <Button variant="outline" onClick={refresh} disabled={isLoading}>
          <RefreshCcw
            className={cn("size-4", isLoading && "animate-spin")}
          />
          Refresh
        </Button>
      </section>

      {flash && (
        <div className="mb-4 rounded-lg border border-vn-accent/30 bg-vn-accent-soft px-4 py-3 text-sm text-vn-text">
          {flash}
        </div>
      )}
      {error && (
        <div className="mb-4 rounded-lg border border-vn-danger/30 bg-vn-danger-soft px-4 py-3 text-sm text-vn-danger">
          {error}
        </div>
      )}

      <div className="mb-3 flex items-center">
        <div className="relative w-full max-w-sm">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-vn-text-4" />
          <Input
            type="search"
            placeholder="Search domains..."
            className="pl-9"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
      </div>

      {!isLoading && domains.length === 0 && !error ? (
        <Card>
          <CardContent className="flex flex-col items-center gap-3 py-14 text-center">
            <div className="flex size-11 items-center justify-center rounded-full bg-vn-accent-soft text-vn-accent">
              <ShieldCheck className="size-5" />
            </div>
            <div>
              <p className="text-sm font-medium text-vn-text">No domains yet</p>
              <p className="mt-1 max-w-md text-xs text-vn-text-3">
                Customers connect their domains through your API key. Domains
                they add will appear here for you to monitor and verify.
              </p>
            </div>
          </CardContent>
        </Card>
      ) : (
        <section className="overflow-hidden rounded-lg border border-vn-hairline bg-vn-surface shadow-vn-soft">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Domain</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="hidden md:table-cell">IP</TableHead>
                <TableHead className="hidden lg:table-cell">Added</TableHead>
                <TableHead className="w-[100px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading && filtered.length === 0 &&
                [0, 1, 2].map((i) => (
                  <TableRow key={`skeleton-${i}`}>
                    <TableCell colSpan={5}>
                      <div className="h-6 w-full animate-pulse rounded bg-vn-surface-2" />
                    </TableCell>
                  </TableRow>
                ))}
              {filtered.map((d) => (
                <TableRow key={d.id}>
                  <TableCell className="font-medium text-vn-text">
                    {d.domain_name}
                  </TableCell>
                  <TableCell>
                    <StatusBadge status={d.status} />
                  </TableCell>
                  <TableCell className="hidden font-mono text-xs text-vn-text-3 md:table-cell">
                    {d.ip || "—"}
                  </TableCell>
                  <TableCell className="hidden text-vn-text-3 lg:table-cell">
                    {new Date(d.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell>
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => handleVerify(d.domain_name)}
                      disabled={verifying === d.domain_name}
                    >
                      {verifying === d.domain_name ? "…" : "Verify"}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
              {!isLoading && filtered.length === 0 && query && (
                <TableRow>
                  <TableCell colSpan={5} className="py-8 text-center text-sm text-vn-text-3">
                    No domains match "{query}".
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </section>
      )}
    </div>
  )
}
