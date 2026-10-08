import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Switch } from "@/components/ui/switch"
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  AlertTriangle,
  Check,
  Loader2,
  Plus,
  Server,
  Sparkles,
  Star,
  Trash2,
} from "lucide-react"
import { useSmtpConfigs } from "@/hooks/useSmtpConfigs"
import { useMailBranding } from "@/hooks/useMailBranding"
import type {
  Branding,
  BrandingPayload,
  SmtpConfig,
  SmtpConfigPayload,
  SmtpEncryption,
} from "@/lib/mail/types"

export function MailSettingsFeature() {
  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-semibold text-vn-text">Mail settings</h1>
        <p className="mt-1 text-sm text-vn-text-3">
          Sender credentials and the branding applied to outgoing campaigns.
        </p>
      </div>

      <Tabs defaultValue="senders">
        <TabsList>
          <TabsTrigger value="senders">Senders</TabsTrigger>
          <TabsTrigger value="branding">Branding</TabsTrigger>
        </TabsList>
        <TabsContent value="senders">
          <SendersPanel />
        </TabsContent>
        <TabsContent value="branding">
          <BrandingPanel />
        </TabsContent>
      </Tabs>
    </div>
  )
}

const EMPTY_FORM: SmtpConfigPayload = {
  label: "",
  from_email: "",
  from_name: "",
  username: "",
  password: "",
  host: "",
  port: 587,
  encryption: "starttls",
  is_default: false,
  active: true,
}

function SendersPanel() {
  const {
    configs,
    defaultId,
    isLoading,
    error,
    create,
    update,
    setDefault,
    remove,
    seed,
  } = useSmtpConfigs()

  const [isOpen, setIsOpen] = useState(false)
  const [editing, setEditing] = useState<SmtpConfig | null>(null)
  const [form, setForm] = useState<SmtpConfigPayload>(EMPTY_FORM)
  const [formError, setFormError] = useState("")
  const [isSaving, setIsSaving] = useState(false)
  const [isSeeding, setIsSeeding] = useState(false)

  const set = <K extends keyof SmtpConfigPayload>(
    key: K,
    value: SmtpConfigPayload[K],
  ) => setForm((prev) => ({ ...prev, [key]: value }))

  const openNew = () => {
    setEditing(null)
    setForm(EMPTY_FORM)
    setFormError("")
    setIsOpen(true)
  }

  const openEdit = (config: SmtpConfig) => {
    setEditing(config)
    setForm({
      label: config.label,
      from_email: config.from_email,
      from_name: config.from_name ?? "",
      username: config.username,
      password: "",
      host: config.host,
      port: config.port,
      encryption: config.encryption,
      is_default: config.is_default,
      active: config.active,
    })
    setFormError("")
    setIsOpen(true)
  }

  const handleSave = async () => {
    if (!form.label.trim() || !form.from_email.trim() || !form.host.trim()) {
      setFormError("Label, from address and host are required")
      return
    }
    if (!editing && !form.password) {
      setFormError("A password is required for a new sender")
      return
    }

    setIsSaving(true)
    setFormError("")
    try {
      if (editing) {
        await update(editing.id, form)
      } else {
        await create(form)
      }
      setIsOpen(false)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Could not save sender")
    } finally {
      setIsSaving(false)
    }
  }

  const handleSeed = async () => {
    setIsSeeding(true)
    try {
      await seed()
    } catch {
      // Surfaced by the hook's error state on refresh.
    } finally {
      setIsSeeding(false)
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm text-vn-text-3">
          Keep as many senders as you need — one per provider or sending domain.
          Campaigns use your default unless you pick another.
        </p>
        <div className="flex items-center gap-2">
          {configs.length === 0 && (
            <Button variant="outline" onClick={() => void handleSeed()} disabled={isSeeding}>
              {isSeeding ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <Sparkles className="size-4" />
              )}
              Use vnytros default
            </Button>
          )}
          <Button onClick={openNew}>
            <Plus className="size-4" />
            Add sender
          </Button>
        </div>
      </div>

      {error && (
        <div className="rounded-xl border border-vn-danger/30 bg-vn-danger-soft p-4 text-sm text-vn-danger">
          {error}
        </div>
      )}

      {isLoading && configs.length === 0 ? (
        <div className="flex items-center gap-2 rounded-xl border border-vn-hairline bg-vn-surface p-6 text-sm text-vn-text-3">
          <Loader2 className="size-4 animate-spin" />
          Loading senders…
        </div>
      ) : configs.length === 0 ? (
        <div className="rounded-xl border border-vn-hairline bg-vn-surface p-8 text-center">
          <Server className="mx-auto size-6 text-vn-text-4" />
          <p className="mt-3 text-sm font-medium text-vn-text">No senders yet</p>
          <p className="mt-1 text-sm text-vn-text-3">
            Add SMTP credentials, or import the vnytros default to get going.
          </p>
        </div>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-vn-hairline bg-vn-surface">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Sender</TableHead>
                <TableHead>Host</TableHead>
                <TableHead>Security</TableHead>
                <TableHead>State</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {configs.map((config) => (
                <TableRow key={config.id}>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <p className="font-medium text-vn-text">{config.label}</p>
                      {config.id === defaultId && (
                        <Badge variant="default">Default</Badge>
                      )}
                    </div>
                    <p className="mt-0.5 text-xs text-vn-text-3">
                      {config.from_name
                        ? `${config.from_name} <${config.from_email}>`
                        : config.from_email}
                    </p>
                  </TableCell>
                  <TableCell className="text-vn-text-2">
                    {config.host}
                    <span className="text-vn-text-4">:{config.port}</span>
                  </TableCell>
                  <TableCell className="text-vn-text-2">
                    {config.encryption}
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-col gap-1">
                      <Badge variant={config.active ? "default" : "outline"}>
                        {config.active ? "Active" : "Disabled"}
                      </Badge>
                      {config.last_error && (
                        <span className="flex max-w-xs items-start gap-1 text-xs text-vn-danger">
                          <AlertTriangle className="mt-0.5 size-3 shrink-0" />
                          <span className="break-words">{config.last_error}</span>
                        </span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-1.5">
                      {config.id !== defaultId && (
                        <Button
                          variant="ghost"
                          size="icon"
                          aria-label="Make default"
                          onClick={() => void setDefault(config.id)}
                        >
                          <Star className="size-3.5" />
                        </Button>
                      )}
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => openEdit(config)}
                      >
                        Edit
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label="Delete sender"
                        onClick={() => void remove(config.id)}
                      >
                        <Trash2 className="size-3.5 text-vn-danger" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <Dialog open={isOpen} onOpenChange={setIsOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {editing ? `Edit ${editing.label}` : "Add sender"}
            </DialogTitle>
            <DialogDescription>
              {editing
                ? "Leave the password blank to keep the stored one."
                : "These credentials are used to deliver your campaigns."}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3">
            <div>
              <label className="mb-1.5 block text-sm text-vn-text-2">Label</label>
              <Input
                value={form.label}
                onChange={(e) => set("label", e.target.value)}
                placeholder="Mailgun production"
              />
            </div>

            <div className="grid gap-3 sm:grid-cols-2">
              <div>
                <label className="mb-1.5 block text-sm text-vn-text-2">
                  From address
                </label>
                <Input
                  type="email"
                  value={form.from_email}
                  onChange={(e) => set("from_email", e.target.value)}
                  placeholder="hello@yourdomain.com"
                />
              </div>
              <div>
                <label className="mb-1.5 block text-sm text-vn-text-2">
                  From name
                </label>
                <Input
                  value={form.from_name ?? ""}
                  onChange={(e) => set("from_name", e.target.value)}
                  placeholder="Optional"
                />
              </div>
            </div>

            <div className="grid gap-3 sm:grid-cols-2">
              <div>
                <label className="mb-1.5 block text-sm text-vn-text-2">Host</label>
                <Input
                  value={form.host}
                  onChange={(e) => set("host", e.target.value)}
                  placeholder="smtp.mailgun.org"
                />
              </div>
              <div>
                <label className="mb-1.5 block text-sm text-vn-text-2">Port</label>
                <Input
                  type="number"
                  value={form.port}
                  onChange={(e) => set("port", Number(e.target.value))}
                />
              </div>
            </div>

            <div className="grid gap-3 sm:grid-cols-2">
              <div>
                <label className="mb-1.5 block text-sm text-vn-text-2">
                  SMTP username
                </label>
                <Input
                  value={form.username ?? ""}
                  onChange={(e) => set("username", e.target.value)}
                  placeholder="Defaults to the from address"
                />
              </div>
              <div>
                <label className="mb-1.5 block text-sm text-vn-text-2">
                  Security
                </label>
                <select
                  value={form.encryption}
                  onChange={(e) =>
                    set("encryption", e.target.value as SmtpEncryption)
                  }
                  className="w-full rounded-lg border border-vn-hairline-2 bg-vn-surface-2 px-3 py-2 text-sm text-vn-text focus:outline-none focus:ring-2 focus:ring-vn-accent/35"
                >
                  <option value="starttls">STARTTLS (587)</option>
                  <option value="ssl">SSL/TLS (465)</option>
                  <option value="none">None</option>
                </select>
              </div>
            </div>

            <div>
              <label className="mb-1.5 block text-sm text-vn-text-2">
                Password
              </label>
              <Input
                type="password"
                value={form.password ?? ""}
                onChange={(e) => set("password", e.target.value)}
                placeholder={editing ? "Unchanged" : "SMTP password or API key"}
              />
            </div>

            <div className="flex items-center justify-between rounded-lg border border-vn-hairline bg-vn-surface-2 px-3 py-2.5">
              <div>
                <p className="text-sm text-vn-text">Make default</p>
                <p className="text-xs text-vn-text-3">
                  Used when a campaign does not name a sender.
                </p>
              </div>
              <Switch
                checked={form.is_default ?? false}
                onChange={(e) => set("is_default", e.target.checked)}
              />
            </div>

            <div className="flex items-center justify-between rounded-lg border border-vn-hairline bg-vn-surface-2 px-3 py-2.5">
              <div>
                <p className="text-sm text-vn-text">Active</p>
                <p className="text-xs text-vn-text-3">
                  Disabled senders cannot be selected for a send.
                </p>
              </div>
              <Switch
                checked={form.active ?? true}
                onChange={(e) => set("active", e.target.checked)}
              />
            </div>

            {formError && <p className="text-sm text-vn-danger">{formError}</p>}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setIsOpen(false)}>
              Cancel
            </Button>
            <Button onClick={() => void handleSave()} disabled={isSaving}>
              {isSaving && <Loader2 className="size-4 animate-spin" />}
              {editing ? "Save changes" : "Add sender"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function BrandingPanel() {
  const { branding, isLoading, error, update } = useMailBranding()

  if (isLoading && !branding) {
    return (
      <div className="flex items-center gap-2 rounded-xl border border-vn-hairline bg-vn-surface p-6 text-sm text-vn-text-3">
        <Loader2 className="size-4 animate-spin" />
        Loading branding…
      </div>
    )
  }

  return (
    <>
      {error && (
        <div className="mb-4 rounded-xl border border-vn-danger/30 bg-vn-danger-soft p-4 text-sm text-vn-danger">
          {error}
        </div>
      )}
      {branding && (
        // Remounting on the loaded values seeds the form from props without an
        // effect that mirrors server state into local state.
        <BrandingForm
          key={branding.watermark_image_url}
          branding={branding}
          onSave={update}
        />
      )}
    </>
  )
}

function BrandingForm({
  branding,
  onSave,
}: {
  branding: Branding
  onSave: (payload: BrandingPayload) => Promise<Branding>
}) {
  const [showWatermark, setShowWatermark] = useState(branding.show_watermark)
  const [imageUrl, setImageUrl] = useState(branding.watermark_image_url)
  const [linkUrl, setLinkUrl] = useState(branding.watermark_link_url)
  const [label, setLabel] = useState(branding.watermark_label)
  const [saveError, setSaveError] = useState("")
  const [saved, setSaved] = useState(false)
  const [isSaving, setIsSaving] = useState(false)

  const handleSave = async () => {
    setIsSaving(true)
    setSaveError("")
    setSaved(false)
    try {
      await onSave({
        show_watermark: showWatermark,
        watermark_image_url: imageUrl,
        watermark_link_url: linkUrl,
        watermark_label: label,
      })
      setSaved(true)
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : "Could not save branding")
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <div className="space-y-4">
      <div className="rounded-xl border border-vn-hairline bg-vn-surface p-5">
        <div className="space-y-4">
          <div className="flex items-center justify-between gap-4">
            <div>
              <p className="text-sm font-medium text-vn-text">Show watermark</p>
              <p className="mt-0.5 text-xs text-vn-text-3">
                Appended to the footer of every campaign email.
              </p>
            </div>
            <Switch
              checked={showWatermark}
              onChange={(e) => setShowWatermark(e.target.checked)}
            />
          </div>

          <div>
            <label className="mb-1.5 block text-sm text-vn-text-2">
              Watermark image
            </label>
            <select
              value={imageUrl}
              onChange={(e) => setImageUrl(e.target.value)}
              className="w-full rounded-lg border border-vn-hairline-2 bg-vn-surface-2 px-3 py-2 text-sm text-vn-text disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-vn-accent/35"
            >
              {(branding.allowed_image_urls ?? [imageUrl])
                .filter(Boolean)
                .map((url) => (
                  <option key={url} value={url}>
                    {url.split("/").pop()}
                  </option>
                ))}
            </select>
            <p className="mt-1.5 text-xs text-vn-text-3">
              Custom uploads are not available yet, so this lists the hosted
              Vnytros assets.
            </p>
          </div>

          {imageUrl && (
            <div className="flex items-center gap-3 rounded-lg border border-vn-hairline bg-vn-surface-2 p-3">
              <img
                src={imageUrl}
                alt="Watermark preview"
                className="h-8 w-auto object-contain"
              />
              <span className="text-xs text-vn-text-3">Preview</span>
            </div>
          )}

          <div className="grid gap-3 sm:grid-cols-2">
            <div>
              <label className="mb-1.5 block text-sm text-vn-text-2">
                Link target
              </label>
              <Input
                value={linkUrl}
                onChange={(e) => setLinkUrl(e.target.value)}
                placeholder="https://your-site.com"
              />
            </div>
            <div>
              <label className="mb-1.5 block text-sm text-vn-text-2">
                Footer text
              </label>
              <Input
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                placeholder="Sent with Vnytros"
              />
            </div>
          </div>

          {saveError && <p className="text-sm text-vn-danger">{saveError}</p>}

          <div className="flex items-center gap-2">
            <Button onClick={() => void handleSave()} disabled={isSaving}>
              {isSaving && <Loader2 className="size-4 animate-spin" />}
              Save branding
            </Button>
            {saved && (
              <span className="flex items-center gap-1 text-sm text-vn-success">
                <Check className="size-4" />
                Saved
              </span>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
