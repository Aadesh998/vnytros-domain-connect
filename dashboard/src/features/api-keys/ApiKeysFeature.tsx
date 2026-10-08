import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import {
  Key,
  MoreHorizontal,
  Plus,
  ShieldCheck,
  Webhook,
  History,
  Copy,
  Check,
  AlertTriangle,
  ExternalLink,
  Ban,
  Activity,
  Trash2,
  Loader2,
  Pencil,
  Filter,
  X,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { useApiKeys } from "@/hooks/useApiKeys"
import { useWebhookLogs } from "@/hooks/useWebhookLogs"
import type {
  ApiKey,
  ApiKeyStatus,
  WebhookLogFilters,
  WebhookStatus,
} from "@/lib/api-keys/types"

export function ApiKeysFeature() {
  const {
    keys,
    isLoading: isKeysLoading,
    error: keysError,
    hasMore: keysHasMore,
    loadMore: loadMoreKeys,
    createKey,
    updateKey,
    deleteKey,
  } = useApiKeys()

  const {
    logs,
    isLoading: isLogsLoading,
    error: logsError,
    hasMore: logsHasMore,
    loadMore: loadMoreLogs,
    applyFilters,
  } = useWebhookLogs()

  // Create dialog state
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [createWebhookUrl, setCreateWebhookUrl] = useState("")
  const [createError, setCreateError] = useState("")
  const [newKey, setNewKey] = useState<string | null>(null)
  const [hasCopied, setHasCopied] = useState(false)
  const [isCreating, setIsCreating] = useState(false)

  // Edit webhook dialog state
  const [editingKey, setEditingKey] = useState<ApiKey | null>(null)
  const [editWebhookUrl, setEditWebhookUrl] = useState("")
  const [editError, setEditError] = useState("")
  const [isUpdating, setIsUpdating] = useState(false)

  // Webhook log filter form state (applied on submit)
  const [filterStatus, setFilterStatus] = useState<WebhookStatus | "">("")
  const [filterEvent, setFilterEvent] = useState("")
  const [filterApiKeyId, setFilterApiKeyId] = useState("")
  const [filterResponseStatus, setFilterResponseStatus] = useState("")
  const [filterFrom, setFilterFrom] = useState("")
  const [filterTo, setFilterTo] = useState("")

  const openCreateDialog = () => {
    setCreateWebhookUrl("")
    setCreateError("")
    setNewKey(null)
    setHasCopied(false)
    setIsCreateOpen(true)
  }

  const closeCreateDialog = () => {
    if (isCreating) return
    setIsCreateOpen(false)
    setNewKey(null)
    setCreateWebhookUrl("")
    setCreateError("")
    setHasCopied(false)
  }

  const handleCreateKey = async () => {
    const url = createWebhookUrl.trim()
    if (!url) {
      setCreateError("Webhook URL is required.")
      return
    }
    if (!url.startsWith("https://")) {
      setCreateError("Endpoint URL must use HTTPS.")
      return
    }
    setCreateError("")
    setIsCreating(true)
    try {
      const generated = await createKey(url)
      setNewKey(generated)
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : "Failed to create key")
    } finally {
      setIsCreating(false)
    }
  }

  const openEditDialog = (key: ApiKey) => {
    setEditingKey(key)
    setEditWebhookUrl(key.web_hook)
    setEditError("")
  }

  const closeEditDialog = () => {
    if (isUpdating) return
    setEditingKey(null)
    setEditWebhookUrl("")
    setEditError("")
  }

  const handleSaveEdit = async () => {
    if (!editingKey) return
    const url = editWebhookUrl.trim()
    if (!url) {
      setEditError("Webhook URL is required.")
      return
    }
    if (!url.startsWith("https://")) {
      setEditError("Endpoint URL must use HTTPS.")
      return
    }
    setIsUpdating(true)
    try {
      await updateKey(editingKey.id, { web_hook: url })
      setEditingKey(null)
      setEditWebhookUrl("")
      setEditError("")
    } catch (err) {
      setEditError(err instanceof Error ? err.message : "Failed to update webhook URL")
    } finally {
      setIsUpdating(false)
    }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    setHasCopied(true)
    setTimeout(() => setHasCopied(false), 2000)
  }

  const toggleKeyStatus = async (id: number, currentStatus: ApiKeyStatus) => {
    const next: ApiKeyStatus = currentStatus === "active" ? "disable" : "active"
    try {
      await updateKey(id, { status: next })
    } catch (err) {
      console.error(err)
    }
  }

  const handleDeleteKey = async (id: number) => {
    if (!confirm("Are you sure you want to revoke this API key? This action cannot be undone.")) return
    try {
      await deleteKey(id)
    } catch (err) {
      console.error(err)
    }
  }

  const handleApplyFilters = () => {
    const next: WebhookLogFilters = {}
    if (filterStatus) next.status = filterStatus
    if (filterEvent.trim()) next.event = filterEvent.trim()
    const apiKeyIdNum = Number(filterApiKeyId)
    if (filterApiKeyId && Number.isFinite(apiKeyIdNum) && apiKeyIdNum > 0) {
      next.api_key_id = apiKeyIdNum
    }
    const responseStatusNum = Number(filterResponseStatus)
    if (filterResponseStatus && Number.isFinite(responseStatusNum) && responseStatusNum > 0) {
      next.response_status = responseStatusNum
    }
    if (filterFrom) next.from = filterFrom
    if (filterTo) next.to = filterTo
    applyFilters(next)
  }

  const handleClearFilters = () => {
    setFilterStatus("")
    setFilterEvent("")
    setFilterApiKeyId("")
    setFilterResponseStatus("")
    setFilterFrom("")
    setFilterTo("")
    applyFilters({})
  }

  const hasActiveFilters = Boolean(
    filterStatus ||
      filterEvent ||
      filterApiKeyId ||
      filterResponseStatus ||
      filterFrom ||
      filterTo,
  )

  return (
    <div>
      <section className="mb-5 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <span className="vn-eyebrow mb-2">developer</span>
          <h1 className="mt-2 text-3xl font-[540] tracking-[-0.035em] text-vn-text md:text-4xl">
            Developer Settings
          </h1>
          <p className="mt-2 max-w-2xl text-vn-text-2">
            Manage your API keys, configure webhooks, and monitor delivery logs for your Vnytros integration.
          </p>
        </div>
      </section>

      <Tabs defaultValue="api-keys" className="w-full">
        <TabsList className="mb-4">
          <TabsTrigger value="api-keys" className="gap-2">
            <Key className="size-3.5" />
            API Keys
          </TabsTrigger>
          <TabsTrigger value="webhooks" className="gap-2">
            <Webhook className="size-3.5" />
            Webhooks
          </TabsTrigger>
          <TabsTrigger value="logs" className="gap-2">
            <History className="size-3.5" />
            Webhook Logs
          </TabsTrigger>
        </TabsList>

        <TabsContent value="api-keys" className="space-y-4">
          <div className="flex justify-end">
            <Button onClick={openCreateDialog}>
              <Plus className="size-4" />
              Create secret key
            </Button>
          </div>

          {keysError && (
            <div className="rounded-xl border border-vn-danger/30 bg-vn-danger-soft p-4 text-sm text-vn-danger">
              {keysError}
            </div>
          )}

          <section className="rounded-lg border border-vn-hairline bg-vn-surface p-4 shadow-vn-soft">
            <div className="flex items-start gap-3">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-md bg-vn-accent-soft text-vn-accent">
                <ShieldCheck className="size-5" />
              </div>
              <div>
                <h2 className="font-semibold text-vn-text">Key visibility & security</h2>
                <p className="mt-1 max-w-xl text-sm text-vn-text-3">
                  Secret keys are only displayed once upon creation. If you lose a key, you'll need to revoke it and create a new one.
                </p>
              </div>
            </div>
          </section>

          <section className="overflow-hidden rounded-lg border border-vn-hairline bg-vn-surface shadow-vn-soft">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>Secret key (masked)</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="hidden md:table-cell">Domains</TableHead>
                  <TableHead className="w-[56px]" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {isKeysLoading && keys.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="h-24 text-center">
                      <Loader2 className="mx-auto size-6 animate-spin text-vn-accent" />
                    </TableCell>
                  </TableRow>
                ) : keys.length > 0 ? (
                  keys.map((key) => (
                    <TableRow key={key.id}>
                      <TableCell className="font-medium text-vn-text">
                        {key.id}
                      </TableCell>
                      <TableCell className="font-mono text-xs text-vn-text-3">
                        {key.api_key}
                      </TableCell>
                      <TableCell>
                        <Badge variant={key.status === "active" ? "default" : "secondary"}>
                          {key.status}
                        </Badge>
                      </TableCell>
                      <TableCell className="hidden text-vn-text-3 md:table-cell">
                        {key.domain_count}
                      </TableCell>
                      <TableCell>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="icon" className="size-8">
                              <MoreHorizontal className="size-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem className="gap-2" onClick={() => copyToClipboard(String(key.id))}>
                              <Copy className="size-3.5" /> Copy ID
                            </DropdownMenuItem>
                            <DropdownMenuItem className="gap-2" onClick={() => openEditDialog(key)}>
                              <Pencil className="size-3.5" /> Edit webhook URL
                            </DropdownMenuItem>
                            <DropdownMenuItem className="gap-2" onClick={() => toggleKeyStatus(key.id, key.status)}>
                              {key.status === "active" ? (
                                <><Ban className="size-3.5" /> Disable key</>
                              ) : (
                                <><Activity className="size-3.5" /> Enable key</>
                              )}
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              className="gap-2 text-vn-danger focus:text-vn-danger focus:bg-vn-danger-soft"
                              onClick={() => handleDeleteKey(key.id)}
                            >
                              <Trash2 className="size-3.5" /> Revoke key
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={5} className="h-24 text-center text-vn-text-4">
                      No API keys found.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>

            {keysHasMore && (
              <div className="flex justify-center border-t border-vn-hairline p-4">
                <Button variant="ghost" size="sm" onClick={loadMoreKeys} disabled={isKeysLoading}>
                  {isKeysLoading ? <Loader2 className="mr-2 size-3 animate-spin" /> : null}
                  Load more keys
                </Button>
              </div>
            )}
          </section>
        </TabsContent>

        <TabsContent value="webhooks" className="space-y-4">
          <div className="flex justify-end">
            <Button onClick={openCreateDialog}>
              <Plus className="size-4" />
              Add webhook
            </Button>
          </div>

          <section className="overflow-hidden rounded-lg border border-vn-hairline bg-vn-surface shadow-vn-soft">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Endpoint URL</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="hidden md:table-cell">Linked Key ID</TableHead>
                  <TableHead className="w-[56px]" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {isKeysLoading && keys.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="h-24 text-center">
                      <Loader2 className="mx-auto size-6 animate-spin text-vn-accent" />
                    </TableCell>
                  </TableRow>
                ) : keys.filter((k) => k.web_hook).length > 0 ? (
                  keys
                    .filter((k) => k.web_hook)
                    .map((key) => (
                      <TableRow key={key.id}>
                        <TableCell className="font-medium text-vn-text">
                          <div className="flex items-center gap-2">
                            <Webhook className="size-4 text-vn-text-4" />
                            {key.web_hook}
                          </div>
                        </TableCell>
                        <TableCell>
                          <Badge variant={key.status === "active" ? "default" : "secondary"}>
                            {key.status}
                          </Badge>
                        </TableCell>
                        <TableCell className="hidden md:table-cell text-vn-text-3">
                          {key.id}
                        </TableCell>
                        <TableCell>
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" size="icon" className="size-8">
                                <MoreHorizontal className="size-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem className="gap-2" onClick={() => openEditDialog(key)}>
                                <Pencil className="size-3.5" /> Edit URL
                              </DropdownMenuItem>
                              <DropdownMenuItem className="gap-2" onClick={() => toggleKeyStatus(key.id, key.status)}>
                                {key.status === "active" ? (
                                  <><Ban className="size-3.5" /> Disable webhook</>
                                ) : (
                                  <><Activity className="size-3.5" /> Enable webhook</>
                                )}
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                className="gap-2 text-vn-danger focus:text-vn-danger focus:bg-vn-danger-soft"
                                onClick={() => handleDeleteKey(key.id)}
                              >
                                <Trash2 className="size-3.5" /> Delete
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </TableCell>
                      </TableRow>
                    ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={4} className="h-24 text-center text-vn-text-4">
                      No webhooks configured.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>

            {keysHasMore && (
              <div className="flex justify-center border-t border-vn-hairline p-4">
                <Button variant="ghost" size="sm" onClick={loadMoreKeys} disabled={isKeysLoading}>
                  {isKeysLoading ? <Loader2 className="mr-2 size-3 animate-spin" /> : null}
                  Load more
                </Button>
              </div>
            )}
          </section>
        </TabsContent>

        <TabsContent value="logs" className="space-y-4">
          <section className="rounded-lg border border-vn-hairline bg-vn-surface p-4 shadow-vn-soft">
            <div className="mb-3 flex items-center gap-2">
              <Filter className="size-4 text-vn-text-3" />
              <h3 className="text-sm font-semibold text-vn-text">Filter logs</h3>
              {hasActiveFilters && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleClearFilters}
                  className="ml-auto h-7 gap-1 px-2 text-xs"
                >
                  <X className="size-3" /> Clear
                </Button>
              )}
            </div>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
              <div className="grid gap-1.5">
                <label htmlFor="filter-status" className="text-[11px] font-mono uppercase tracking-widest text-vn-text-3">
                  Status
                </label>
                <select
                  id="filter-status"
                  value={filterStatus}
                  onChange={(e) => setFilterStatus(e.target.value as WebhookStatus | "")}
                  className="flex h-[42px] w-full rounded-[10px] border border-vn-hairline-2 bg-vn-bg-2 px-3.5 py-2 text-sm text-vn-text shadow-sm transition-colors focus-visible:border-vn-accent/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-vn-accent/25"
                >
                  <option value="">Any</option>
                  <option value="pending">Pending</option>
                  <option value="delivered">Delivered</option>
                  <option value="failed">Failed</option>
                </select>
              </div>
              <div className="grid gap-1.5">
                <label htmlFor="filter-event" className="text-[11px] font-mono uppercase tracking-widest text-vn-text-3">
                  Event
                </label>
                <Input
                  id="filter-event"
                  placeholder="domain.verified"
                  value={filterEvent}
                  onChange={(e) => setFilterEvent(e.target.value)}
                />
              </div>
              <div className="grid gap-1.5">
                <label htmlFor="filter-api-key-id" className="text-[11px] font-mono uppercase tracking-widest text-vn-text-3">
                  API key ID
                </label>
                <Input
                  id="filter-api-key-id"
                  type="number"
                  inputMode="numeric"
                  placeholder="12"
                  value={filterApiKeyId}
                  onChange={(e) => setFilterApiKeyId(e.target.value)}
                />
              </div>
              <div className="grid gap-1.5">
                <label htmlFor="filter-response-status" className="text-[11px] font-mono uppercase tracking-widest text-vn-text-3">
                  Response status
                </label>
                <Input
                  id="filter-response-status"
                  type="number"
                  inputMode="numeric"
                  placeholder="200"
                  value={filterResponseStatus}
                  onChange={(e) => setFilterResponseStatus(e.target.value)}
                />
              </div>
              <div className="grid gap-1.5">
                <label htmlFor="filter-from" className="text-[11px] font-mono uppercase tracking-widest text-vn-text-3">
                  From
                </label>
                <Input
                  id="filter-from"
                  type="date"
                  value={filterFrom}
                  onChange={(e) => setFilterFrom(e.target.value)}
                />
              </div>
              <div className="grid gap-1.5">
                <label htmlFor="filter-to" className="text-[11px] font-mono uppercase tracking-widest text-vn-text-3">
                  To
                </label>
                <Input
                  id="filter-to"
                  type="date"
                  value={filterTo}
                  onChange={(e) => setFilterTo(e.target.value)}
                />
              </div>
            </div>
            <div className="mt-3 flex justify-end">
              <Button size="sm" onClick={handleApplyFilters} disabled={isLogsLoading}>
                {isLogsLoading ? <Loader2 className="mr-2 size-3 animate-spin" /> : null}
                Apply filters
              </Button>
            </div>
          </section>

          {logsError && (
            <div className="rounded-xl border border-vn-danger/30 bg-vn-danger-soft p-4 text-sm text-vn-danger">
              {logsError}
            </div>
          )}

          <section className="overflow-hidden rounded-lg border border-vn-hairline bg-vn-surface shadow-vn-soft">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Timestamp</TableHead>
                  <TableHead>Endpoint</TableHead>
                  <TableHead>Event</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="w-[56px]" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLogsLoading && logs.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="h-24 text-center">
                      <Loader2 className="mx-auto size-6 animate-spin text-vn-accent" />
                    </TableCell>
                  </TableRow>
                ) : logs.length > 0 ? (
                  logs.map((log) => (
                    <TableRow key={log.id}>
                      <TableCell className="text-vn-text-3 text-xs">
                        {new Date(log.created_at).toLocaleString()}
                      </TableCell>
                      <TableCell className="max-w-[200px] truncate font-medium text-xs">
                        {log.web_hook}
                      </TableCell>
                      <TableCell>
                        <code className="text-[10px] bg-vn-surface-2 px-1.5 py-0.5 rounded text-vn-text-2">
                          {log.event}
                        </code>
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant={log.status === "delivered" ? "default" : "destructive"}
                          className="text-[10px] px-1.5 py-0"
                        >
                          {log.response_status} {log.status}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Button variant="ghost" size="icon" className="size-8">
                          <ExternalLink className="size-4" />
                          <span className="sr-only">View details</span>
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={5} className="h-24 text-center text-vn-text-4">
                      No webhook logs available.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>

            {logsHasMore && (
              <div className="flex justify-center border-t border-vn-hairline p-4">
                <Button variant="ghost" size="sm" onClick={loadMoreLogs} disabled={isLogsLoading}>
                  {isLogsLoading ? <Loader2 className="mr-2 size-3 animate-spin" /> : null}
                  Load more logs
                </Button>
              </div>
            )}
          </section>
        </TabsContent>
      </Tabs>

      {/* Create Key Dialog */}
      <Dialog open={isCreateOpen} onOpenChange={(open) => !open && closeCreateDialog()}>
        <DialogContent className="sm:max-w-[425px]">
          {!newKey ? (
            <>
              <DialogHeader>
                <DialogTitle>Create API key</DialogTitle>
                <DialogDescription>
                  Enter the webhook URL where events should be delivered. A new API key will be generated.
                </DialogDescription>
              </DialogHeader>
              <div className="grid gap-5 py-4">
                <div className="grid gap-2">
                  <label htmlFor="create-webhook" className="text-xs font-mono uppercase tracking-widest text-vn-text-3">
                    Webhook URL
                  </label>
                  <Input
                    id="create-webhook"
                    placeholder="https://api.yourdomain.com/webhook"
                    value={createWebhookUrl}
                    onChange={(e) => {
                      setCreateWebhookUrl(e.target.value)
                      if (createError) setCreateError("")
                    }}
                    className={cn(createError && "border-vn-danger/50 focus-visible:ring-vn-danger/25")}
                  />
                  {createError && (
                    <p className="text-[12px] text-vn-danger font-medium">{createError}</p>
                  )}
                </div>
              </div>
              <DialogFooter className="gap-2 sm:space-x-0">
                <Button variant="outline" onClick={closeCreateDialog} disabled={isCreating}>
                  Cancel
                </Button>
                <Button onClick={handleCreateKey} disabled={!createWebhookUrl.trim() || isCreating}>
                  {isCreating ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
                  Create key
                </Button>
              </DialogFooter>
            </>
          ) : (
            <>
              <DialogHeader>
                <div className="mx-auto mb-4 flex size-12 items-center justify-center rounded-full bg-vn-accent-soft text-vn-accent">
                  <Check className="size-6" />
                </div>
                <DialogTitle className="text-center">Key created successfully</DialogTitle>
                <DialogDescription className="text-center">
                  Copy this key and save it somewhere safe. You won't be able to see it again.
                </DialogDescription>
              </DialogHeader>
              <div className="py-4">
                <div className="relative">
                  <Input readOnly value={newKey} className="pr-12 font-mono text-xs" />
                  <Button
                    size="icon"
                    variant="ghost"
                    className="absolute right-1 top-1/2 size-8 -translate-y-1/2"
                    onClick={() => copyToClipboard(newKey)}
                  >
                    {hasCopied ? <Check className="size-4 text-vn-success" /> : <Copy className="size-4" />}
                  </Button>
                </div>
                <div className="mt-4 flex items-start gap-3 rounded-lg border border-vn-warn/30 bg-vn-warn-soft p-3 text-[13px] text-vn-warn">
                  <AlertTriangle className="size-4 shrink-0 mt-0.5" />
                  <p>Anyone with this key can access your Vnytros resources. Do not share it in public repositories.</p>
                </div>
              </div>
              <DialogFooter>
                <Button className="w-full" onClick={closeCreateDialog}>Done</Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>

      {/* Edit Webhook URL Dialog */}
      <Dialog open={editingKey !== null} onOpenChange={(open) => !open && closeEditDialog()}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Edit webhook URL</DialogTitle>
            <DialogDescription>
              Update the endpoint where webhook events will be delivered for this API key.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-5 py-4">
            <div className="grid gap-2">
              <label htmlFor="edit-webhook" className="text-xs font-mono uppercase tracking-widest text-vn-text-3">
                Webhook URL
              </label>
              <Input
                id="edit-webhook"
                placeholder="https://api.yourdomain.com/webhook"
                value={editWebhookUrl}
                onChange={(e) => {
                  setEditWebhookUrl(e.target.value)
                  if (editError) setEditError("")
                }}
                className={cn(editError && "border-vn-danger/50 focus-visible:ring-vn-danger/25")}
              />
              {editError && (
                <p className="text-[12px] text-vn-danger font-medium">{editError}</p>
              )}
            </div>
          </div>
          <DialogFooter className="gap-2 sm:space-x-0">
            <Button variant="outline" onClick={closeEditDialog} disabled={isUpdating}>
              Cancel
            </Button>
            <Button
              onClick={handleSaveEdit}
              disabled={
                !editWebhookUrl.trim() ||
                editWebhookUrl.trim() === editingKey?.web_hook ||
                isUpdating
              }
            >
              {isUpdating ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
              Save changes
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
